package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const Address = "localhost:9000"

func main() {
	conn, err := net.Dial("tcp", Address)
	if err != nil {
		fmt.Printf("Could not connect to server at %s. Is main.go running?\n", Address)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("--- REMOTE CONTROL ---")
	fmt.Println("Commands: !w !s !a !d, !turnleft !turnright, !jump, !build, !destroy")
	fmt.Println("Type command and press ENTER.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "exit" {
			break
		}
		if text == "" {
			continue
		}

		fmt.Fprintf(conn, "%s\n", text)
	}
}
