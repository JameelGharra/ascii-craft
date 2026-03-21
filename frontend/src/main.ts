import { type AppConfig, isValidAppConfig } from "./types";
import { Application } from "./application";

const urlConfig = {
    apiBaseUrl: import.meta.env.VITE_API_BASE_URL,
    wsBaseUrl: import.meta.env.VITE_WS_BASE_URL,
};

function checkUrlConfig() {
    if (!urlConfig.apiBaseUrl || !urlConfig.wsBaseUrl) {
        throw new Error("Missing required environment variables. Check your .env file.");
    }
}

const DEFAULT_CONFIG_FETCH_DELAY_MS = 2000;

async function fetchConfigWithRetry(url: string, maxRetries = Infinity, delayMs = DEFAULT_CONFIG_FETCH_DELAY_MS): Promise<AppConfig> {
    const statusEl = document.getElementById("status");
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
            const response = await fetch(url, { cache: "no-store" });
            if (!response.ok) throw new Error(`HTTP error status: ${response.status}`);
            
            const data = await response.json();
            if (!isValidAppConfig(data)) throw new Error("Invalid config format from server");
            
            return data;
        } catch (err) {
            console.warn(`Config fetch attempt ${attempt} failed:`, err);
            if (statusEl) {
                statusEl.innerHTML = `<span class="status-dot status-wait"></span>WAITING FOR SERVER (${attempt}/${maxRetries === Infinity ? '∞' : maxRetries})`;
            }
            if (attempt < maxRetries) {
                await new Promise(resolve => setTimeout(resolve, delayMs));
            }
        }
    }
    throw new Error("Could not reach to server to fetch configuration.");
}

async function bootstrap() {
    checkUrlConfig();

    try {
        const MAX_CONFIG_FETCH_RETRIES = 10;
        const configUrl = `${urlConfig.apiBaseUrl}/api/config`;
        
        // 1. Fetch initial configuration
        const initialConfig = await fetchConfigWithRetry(configUrl, MAX_CONFIG_FETCH_RETRIES);
        
        // 2. Create the reload callback for the orchestrator
        const reloadConfigFn = () => fetchConfigWithRetry(configUrl, 3, 1000);

        // 3. Initialize and start the application
        const app = new Application(initialConfig, `${urlConfig.wsBaseUrl}/ws`, reloadConfigFn);
        app.start();

        // Optional: Attach to window for debugging/teardown
        (window as any).craftApp = app;

    } catch (err) {
        console.error("Initialization failed:", err);
        const statusEl = document.getElementById("status");
        if (statusEl) {
            statusEl.innerHTML = `<span class="status-dot status-offline"></span>INIT FAILED`;
        }
    }
}

// Ignition
bootstrap();