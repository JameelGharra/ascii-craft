import type { AppConfig } from "./types";
import { EventBus, type AppEventMap, type IDisposable } from "./core";
import { ConnectionManager, MessageRouter } from "./network";
import { MetricsCollector, LatencyTracker } from "./metrics";
import { FramePipeline } from "./pipeline";
import { 
    StatusHeader, 
    StatusOverlay, 
    TelemetryPanel, 
    CommandTimeline, 
    ChatUI, 
    InputController 
} from "./ui";

export class Application implements IDisposable {
    private readonly bus = new EventBus<AppEventMap>();
    
    // Infrastructure
    private readonly connectionManager: ConnectionManager;
    private readonly messageRouter: MessageRouter;
    private readonly latencyTracker: LatencyTracker;
    
    // UI Components
    private readonly statusHeader = new StatusHeader();
    private readonly statusOverlay = new StatusOverlay();
    private readonly telemetryPanel = new TelemetryPanel();
    private readonly commandTimeline = new CommandTimeline();
    private readonly chatUI: ChatUI;
    private readonly inputController: InputController;

    // Dynamic Components (recreated on config changes)
    private metricsCollector!: MetricsCollector;
    private framePipeline!: FramePipeline;
    private config: AppConfig;

    // State & Timers
    private isConnected = false;
    private isReceivingFrames = false;
    private lastFrameTime = performance.now();
    private uiIntervalId: number | null = null;
    
    private readonly RENDER_UI_INTERVAL_MS = 250;
    private readonly WATCHDOG_INTERVAL_MS = 1500;
    
    private readonly wsUrl: string;
    private readonly reloadConfigFn: () => Promise<AppConfig>;

    constructor(initialConfig: AppConfig, wsUrl: string, reloadConfigFn: () => Promise<AppConfig>) {
        this.config = initialConfig;
        this.wsUrl = wsUrl;
        this.reloadConfigFn = reloadConfigFn;

        // Instantiate infrastructure injecting the EventBus
        this.connectionManager = new ConnectionManager(this.bus);
        this.messageRouter = new MessageRouter(this.bus);
        this.latencyTracker = new LatencyTracker(this.connectionManager);
        
        // Instantiate UI
        this.chatUI = new ChatUI(this.config.chat.max_messages);
        this.inputController = new InputController(this.bus, this.config);

        this.setupPipeline(this.config);
        this.bindEvents();
    }

    public start(): void {
        this.statusOverlay.showConnecting();
        this.inputController.setSystemDisabled(true, "WAITING FOR SERVER...");
        
        this.uiIntervalId = window.setInterval(this.onTick, this.RENDER_UI_INTERVAL_MS);
        this.connectionManager.connect(this.wsUrl);
    }

    private setupPipeline(newConfig: AppConfig): void {
        this.config = newConfig;
        this.statusHeader.setConfig(newConfig.video.width, newConfig.video.height);
        
        this.metricsCollector = new MetricsCollector(newConfig.video.width, newConfig.video.height);
        
        if (this.framePipeline) {
            this.framePipeline.dispose();
        }
        
        this.framePipeline = new FramePipeline('gameCanvas', newConfig.video.width, newConfig.video.height, this.metricsCollector);
        this.framePipeline.resetSyncState();
        this.inputController.updateConfig(newConfig);
    }

    private bindEvents(): void {
        // Connection Lifecycle
        this.bus.on('connection:connected', this.onConnected);
        this.bus.on('connection:disconnected', this.onDisconnected);
        this.bus.on('connection:reconnecting', this.onReconnecting);
        this.bus.on('connection:error', (err) => console.error("Connection Error:", err));

        // Network Data
        this.bus.on('network:packet', this.onPacket);
        this.bus.on('network:message', (msg) => this.messageRouter.handleMessage(msg));

        // Server Messages
        this.bus.on('server:config_reload', this.onConfigReload);
        this.bus.on('server:vote_result', this.onVoteResult);
        this.bus.on('server:viewers_update', (count) => this.metricsCollector.setViewersCount(count));
        this.bus.on('server:pong', (t) => this.latencyTracker.handlePong(t));
        this.bus.on('server:system_message', (msg) => this.chatUI.appendChat("System", msg, "system"));

        // User Input
        this.bus.on('input:command_valid', this.onCommandValid);
        this.bus.on('input:command_invalid', this.onCommandInvalid);
    }

    // --- Event Handlers (Bound via Arrow Functions to preserve 'this') ---

    private onConnected = (): void => {
        this.isConnected = true;
        this.statusHeader.setConnected();
        this.statusOverlay.showNoSignal();
        this.isReceivingFrames = false;
        this.inputController.setSystemDisabled(true, "WAITING FOR VIDEO FEED...");
        this.framePipeline.resetSyncState();
        this.latencyTracker.start();
    };

    private onDisconnected = (): void => {
        this.isConnected = false;
        this.statusHeader.setDisconnected();
        this.inputController.setSystemDisabled(true, "SYSTEM OFFLINE");
        this.isReceivingFrames = false;
        this.latencyTracker.stop();
    };

    private onReconnecting = (delayMs: number): void => {
        const seconds = Math.round(delayMs / 1000);
        this.statusHeader.setReconnecting(seconds);
        this.statusOverlay.showReconnecting(seconds);
        this.inputController.setSystemDisabled(true, `RECONNECTING IN ${seconds}S...`);
        this.chatUI.appendChat("System", `Connection lost. Retrying in ${seconds}s...`, "system");
    };

    private onPacket = (buffer: ArrayBuffer): void => {
        this.lastFrameTime = performance.now();
        if (!this.isReceivingFrames) {
            this.isReceivingFrames = true;
            this.statusOverlay.hide();
            this.inputController.setSystemDisabled(false);
        }
        this.framePipeline.handlePacket(buffer);
    };

    private onConfigReload = async (): Promise<void> => {
        console.log("Server indicated a configuration change. Reloading...");
        try {
            const newConfig = await this.reloadConfigFn();
            this.setupPipeline(newConfig);
            this.chatUI.appendChat("System", `Game feed updated to ${newConfig.video.width}x${newConfig.video.height}`, "system");
        } catch (err) {
            console.error("Failed to reload config:", err);
        }
    };

    private onVoteResult = (command: string, votes: number): void => {
        this.commandTimeline.addCommand(command, votes);
        this.chatUI.appendChat("Server", `Majority voted for ${command} (${votes} votes)`, "server");
    };

    private onCommandValid = (cmd: string): void => {
        this.connectionManager.send(cmd);
        this.chatUI.appendChat("You", cmd, "user");
    };

    private onCommandInvalid = (cmd: string): void => {
        this.chatUI.appendChat("System", `Invalid command: ${cmd}`, "system");
    };

    private onTick = (): void => {
        if (!this.isConnected) return;

        const now = performance.now();
        if (this.isReceivingFrames && now - this.lastFrameTime > this.WATCHDOG_INTERVAL_MS) {
            this.isReceivingFrames = false;
            this.statusOverlay.showNoSignal();
            this.inputController.setSystemDisabled(true, "NO VIDEO FEED");
            this.chatUI.appendChat("System", "Video feed lost. Waiting for game server...", "warning");
        }

        const stats = this.metricsCollector.getSnapshot();
        this.telemetryPanel.update(stats, this.latencyTracker.getLatency());
    };

    public dispose(): void {
        if (this.uiIntervalId !== null) {
            window.clearInterval(this.uiIntervalId);
            this.uiIntervalId = null;
        }
        
        this.framePipeline.dispose();
        this.latencyTracker.dispose();
        this.connectionManager.dispose();
        this.inputController.dispose();
        this.bus.dispose();
    }
}