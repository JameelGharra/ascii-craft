package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// --- Phase 1: Configuration Structures ---
type AppConfig struct {
	Commands struct {
		Standard      []string `json:"standard"`
		Parameterized map[string]struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"parameterized"`
	} `json:"commands"`
}

var (
	apiURL   = flag.String("api", "http://localhost:8080/api/config", "URL to fetch game config")
	wsURL    = flag.String("ws", "ws://localhost:8080/ws", "WebSocket URL of the Relay")
	numBots  = flag.Int("bots", 1000, "Number of concurrent bot connections")
	rampUpMs = flag.Int("rampup", 10, "Milliseconds to wait between starting each bot")
)

// Simulator encapsulates the application state and behavior
type Simulator struct {
	apiURL            string
	wsURL             string
	numBots           int
	rampUpMs          int
	config            *AppConfig
	parameterizedKeys []string

	botsConnected atomic.Int64
	cmdsSent      atomic.Uint64
	currentTrend  atomic.Value
}

// NewSimulator creates a new simulator instance and pre-calculates reusable data
func NewSimulator(apiURL, wsURL string, numBots, rampUpMs int, cfg *AppConfig) *Simulator {
	// Pre-allocate map keys to prevent GC thrashing inside the hot worker loop
	keys := make([]string, 0, len(cfg.Commands.Parameterized))
	for k := range cfg.Commands.Parameterized {
		keys = append(keys, k)
	}

	sim := &Simulator{
		apiURL:            apiURL,
		wsURL:             wsURL,
		numBots:           numBots,
		rampUpMs:          rampUpMs,
		config:            cfg,
		parameterizedKeys: keys,
	}

	// Initialize Hivemind trend to the first command
	if len(cfg.Commands.Standard) > 0 {
		sim.currentTrend.Store(cfg.Commands.Standard[0])
	}
	return sim
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.Printf("Starting botsim with %d bots targeted at %s...\n", *numBots, *wsURL)

	// Setup Context with Cancellation for Graceful Shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Fetch Configuration with a reliable HTTP client
	config, err := fetchConfig(*apiURL)
	if err != nil {
		log.Fatalf("Failed to fetch config: %v", err)
	}

	if len(config.Commands.Standard) == 0 {
		log.Fatalf("No standard commands found in config!")
	}

	// Encapsulate State
	sim := NewSimulator(*apiURL, *wsURL, *numBots, *rampUpMs, config)

	var wg sync.WaitGroup

	// Start Background Managers
	go sim.trendManager(ctx)
	go sim.statusLogger(ctx)

	// Orchestration (Staggered Ramp-up)
	for i := 0; i < *numBots; i++ {
		wg.Add(1)
		go sim.botWorker(ctx, i, &wg)

		// Wait to prevent Thundering Herd, but allow early exit if cancelled
		select {
		case <-ctx.Done():
			// Break out of the loop early if Ctrl+C is pressed during ramp-up
			break
		case <-time.After(time.Duration(*rampUpMs) * time.Millisecond):
		}
	}

	// Wait for Ctrl+C (SIGINT) or SIGTERM
	<-ctx.Done()
	fmt.Println("\nShutting down gracefully... waiting for bots to disconnect.")
	wg.Wait()
	fmt.Println("All bots disconnected. Exiting.")
}

func fetchConfig(url string) (*AppConfig, error) {
	// Introduce HTTP Client Resiliency
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cfg AppConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// trendManager changes the "favored" command
func (s *Simulator) trendManager(ctx context.Context) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		// Pick a new trend at most every 5 seconds [1, 5] seconds
		sleepDuration := time.Duration(rng.Intn(5)+1) * time.Second

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepDuration):
		}

		// Pick a random standard command to be the new trend
		newTrend := s.config.Commands.Standard[rng.Intn(len(s.config.Commands.Standard))]
		s.currentTrend.Store(newTrend)
	}
}

// statusLogger periodically logs telemetry to stdout
func (s *Simulator) statusLogger(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			connected := s.botsConnected.Load()
			sent := s.cmdsSent.Load()
			trend := s.currentTrend.Load().(string)
			fmt.Printf("\r[Botsim Status] Connected: %4d/%d | Cmds Sent: %8d | Current Trend: %-10s",
				connected, s.numBots, sent, trend)
		}
	}
}

// botWorker handles an individual simulated player connection
func (s *Simulator) botWorker(ctx context.Context, id int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Local RNG to prevent lock contention on global math/rand
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

	conn, _, err := websocket.DefaultDialer.Dial(s.wsURL, nil)
	if err != nil {
		// It's normal for some to fail if max files/sockets is reached
		return
	}
	defer conn.Close()

	s.botsConnected.Add(1)
	defer s.botsConnected.Add(-1)

	// Channel to signal the writer if reader dies
	readDone := make(chan struct{})

	// Reader Goroutine (Zero-Allocation Drainer)
	go func() {
		defer close(readDone)
		for {
			msgType, r, err := conn.NextReader()
			if err != nil {
				return // Disconnected
			}

			// Discard everything (binary frames and text)
			if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
				io.Copy(io.Discard, r)
			}
		}
	}()

	// Setup a reusable timer to avoid massive garbage collection pressure
	// caused by calling time.After in a hot loop.
	sleepDuration := time.Duration(rng.Intn(2000)+500) * time.Millisecond
	timer := time.NewTimer(sleepDuration)
	defer timer.Stop()

	// Writer Loop
	for {
		select {
		case <-ctx.Done():
			// Cleanly close the websocket connection to not anger the Relay
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case <-readDone:
			// Server disconnected us
			return
		case <-timer.C:
		}

		// 15% chance to be completely silent this tick (Burst/Silence mode)
		if rng.Float32() >= 0.15 {
			var cmd string
			roll := rng.Float32()

			if roll < 0.40 {
				// 40% chance to follow the Hivemind Trend
				cmd = s.currentTrend.Load().(string)
			} else if roll < 0.85 {
				// 45% chance to pick a random standard command (Noise)
				cmd = s.config.Commands.Standard[rng.Intn(len(s.config.Commands.Standard))]
			} else {
				// 15% chance to pick a parameterized command (e.g., !slot 2)
				if len(s.parameterizedKeys) > 0 {
					base := s.parameterizedKeys[rng.Intn(len(s.parameterizedKeys))]
					bounds := s.config.Commands.Parameterized[base]
					val := rng.Intn(bounds.Max-bounds.Min+1) + bounds.Min
					cmd = fmt.Sprintf("%s %d", base, val)
				} else {
					cmd = s.config.Commands.Standard[rng.Intn(len(s.config.Commands.Standard))]
				}
			}

			// Send the vote
			err := conn.WriteMessage(websocket.TextMessage, []byte(cmd))
			if err != nil {
				return
			}
			s.cmdsSent.Add(1)
		}

		// Reset the reusable timer for the next iteration
		sleepDuration = time.Duration(rng.Intn(2000)+500) * time.Millisecond
		timer.Reset(sleepDuration)
	}
}
