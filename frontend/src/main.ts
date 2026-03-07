import { type AppConfig, isValidAppConfig } from "./types";
import { ConnectionManager } from "./network/connection-manager";
import { ChatUI } from "./ui/chat-ui";
import { InputController } from "./ui/input-controller";
import { FramePipeline } from "./pipeline/frame-pipeline";

async function fetchConfigWithRetry(url: string, maxRetries = 10, delayMs = 2000): Promise<AppConfig> {
    const statusEl = document.getElementById("status");
    for(let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
            const response = await fetch(url);
            if(!response.ok) throw new Error(`HTTP error status: ${response.status}`);
            const data = await response.json();
            if(!isValidAppConfig(data)) throw new Error("Invalid config format from server");
            
            if (statusEl) statusEl.innerText = "Config loaded, connecting...";
            return data;
        } catch(err) {
            console.warn(`Config fetch attempt ${attempt} failed:`, err);
            if (statusEl) statusEl.innerText = `Waiting for server... (Attempt ${attempt}/${maxRetries})`;
            if (attempt < maxRetries) {
                await new Promise(resolve => setTimeout(resolve, delayMs));
            }
        }
    }
    throw new Error("Could not reach to server to fetch configuration.");
}

async function initApp() {
    try {
        const config = await fetchConfigWithRetry('http://localhost:8080/api/config');
        const chatUI = new ChatUI(config.chat.max_messages);
        const inputController = new InputController(config);
        const framePipeline = new FramePipeline('gameCanvas', config.video.width, config.video.height);
        const connectionManager = new ConnectionManager();

        // events setup
        connectionManager.onConnect = () => {
            chatUI.updateStatus("Connected");
            framePipeline.resetSyncState(); // Resync frames on fresh connect
        };
        
        connectionManager.onDisconnect = () => {
            chatUI.updateStatus("Disconnected");
        };
        
        connectionManager.onReconnectAttempt = (delayMs) => {
            chatUI.updateStatus(`Reconnecting in ${Math.round(delayMs / 1000)}s...`);
            chatUI.appendChat("System", `Connection lost. Retrying in ${Math.round(delayMs / 1000)}s...`, "system");
        };

        connectionManager.onMessage = (msg) => {
            chatUI.appendChat("Server", msg, "server");
        };

        connectionManager.onPacket = (buffer) => {
            framePipeline.handlePacket(buffer);
        };

        inputController.onValidCommand = (cmd) => {
            connectionManager.send(cmd);
            chatUI.appendChat("You", cmd, "user");
        };

        inputController.onInvalidCommand = (cmd) => {
            chatUI.appendChat("System", `Invalid command: ${cmd}`, "system");
        };

        connectionManager.connect("ws://localhost:8080/ws");

    } catch(err) {
        console.error("Initialization failed:", err);
        const statusEl = document.getElementById("status");
        if(statusEl) statusEl.innerText = "Failed to initialize game. Please refresh later.";
    }
}

initApp();