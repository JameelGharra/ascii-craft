import { type AppConfig, isValidAppConfig } from "./types";
import { ConnectionManager } from "./network/connection-manager";
import { ChatUI } from "./ui/chat-ui";
import { InputController } from "./ui/input-controller";
import { FramePipeline } from "./pipeline/frame-pipeline";
import { MetricsCollector } from "./metrics/metrics-collector";
import { LatencyTracker } from "./metrics/latency-tracker";
import { MessageRouter } from "./network/message-router";

// Phase 4 UI Components
import { StatusHeader } from "./ui/status-header";
import { TelemetryPanel } from "./ui/telemetry-panel";
import { CommandTimeline } from "./ui/command-timeline";
import { StatusOverlay } from "./ui/status-overlay";

const urlConfig = {
    apiBaseUrl: import.meta.env.VITE_API_BASE_URL,
    wsBaseUrl: import.meta.env.VITE_WS_BASE_URL,
};

function checkUrlConfig() {
    if (!urlConfig.apiBaseUrl || !urlConfig.wsBaseUrl) {
        throw new Error(
            "Missing required environment variables. Check your .env file."
        );
    }
}

async function fetchConfigWithRetry(url: string, maxRetries = 10, delayMs = 2000): Promise<AppConfig> {
    const statusEl = document.getElementById("status");
    for(let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
            const response = await fetch(url);
            if(!response.ok) throw new Error(`HTTP error status: ${response.status}`);
            const data = await response.json();
            if(!isValidAppConfig(data)) throw new Error("Invalid config format from server");
            
            return data;
        } catch(err) {
            console.warn(`Config fetch attempt ${attempt} failed:`, err);
            if (statusEl) {
                statusEl.innerHTML = `<span class="status-dot status-wait"></span>WAITING FOR SERVER (${attempt}/${maxRetries})`;
            }
            if (attempt < maxRetries) {
                await new Promise(resolve => setTimeout(resolve, delayMs));
            }
        }
    }
    throw new Error("Could not reach to server to fetch configuration.");
}

async function initApp() {
    try {
        const config = await fetchConfigWithRetry(`${urlConfig.apiBaseUrl}/api/config`);
        
        // --- Infrastructure ---
        const connectionManager = new ConnectionManager();
        const metrics = new MetricsCollector(config.video.width, config.video.height);
        const latencyTracker = new LatencyTracker(connectionManager);
        const messageRouter = new MessageRouter();
        const framePipeline = new FramePipeline('gameCanvas', config.video.width, config.video.height, metrics);

        // --- UI Components ---
        const statusHeader = new StatusHeader();
        statusHeader.setConfig(config.video.width, config.video.height);
        const statusOverlay = new StatusOverlay();
        statusOverlay.showConnecting();
        const telemetryPanel = new TelemetryPanel();
        const commandTimeline = new CommandTimeline();
        const chatUI = new ChatUI(config.chat.max_messages);
        const inputController = new InputController(config);
        
        // --- Routing Events ---
        messageRouter.onVote = (event) => {
            commandTimeline.addCommand(event.command, event.votes);
            chatUI.appendChat("Server", `Majority voted for ${event.command} (${event.votes} votes)`, "server");
        };
        messageRouter.onViewers = (count) => {
            metrics.setViewersCount(count);  
        };
        messageRouter.onPong = (originalTime) => {
            latencyTracker.handlePong(originalTime);
        };
        messageRouter.onSystemMessage = (msg) => {
            chatUI.appendChat("System", msg, "system");
        }

        // --- Connection Events ---
        connectionManager.onConnect = () => {
            statusHeader.setConnected();
            statusOverlay.hide();
            framePipeline.resetSyncState(); // Resync frames on fresh connect
            latencyTracker.start();
        };
        
        connectionManager.onDisconnect = () => {
            statusHeader.setDisconnected();
            latencyTracker.stop();
        };
        
        connectionManager.onReconnectAttempt = (delayMs) => {
            const seconds = Math.round(delayMs / 1000);
            statusHeader.setReconnecting(seconds);
            statusOverlay.showReconnecting(seconds);
            chatUI.appendChat("System", `Connection lost. Retrying in ${seconds}s...`, "system");
        };

        connectionManager.onMessage = (msg) => {
            messageRouter.handleMessage(msg);
        };

        connectionManager.onPacket = (buffer) => {
            framePipeline.handlePacket(buffer);
        };

        // --- Input Events ---
        inputController.onValidCommand = (cmd) => {
            connectionManager.send(cmd);
            chatUI.appendChat("You", cmd, "user");
        };

        inputController.onInvalidCommand = (cmd) => {
            chatUI.appendChat("System", `Invalid command: ${cmd}`, "system");
        };

        // --- Render UI Loop ---
        setInterval(() => {
            if (connectionManager['ws']?.readyState === WebSocket.OPEN) {
                const stats = metrics.getSnapshot();
                telemetryPanel.update(stats, latencyTracker.getLatency());
            }
        }, 250); // Updates the panel 4 times a second

        // Start connection!
        connectionManager.connect(`${urlConfig.wsBaseUrl}/ws`);

    } catch(err) {
        console.error("Initialization failed:", err);
        const statusEl = document.getElementById("status");
        if(statusEl) {
            statusEl.innerHTML = `<span class="status-dot status-offline"></span>INIT FAILED`;
        }
    }
}

checkUrlConfig();
initApp();