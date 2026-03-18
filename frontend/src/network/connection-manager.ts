import { type AppEventMap, EventBus, type IDisposable } from "../core";

export class ConnectionManager implements IDisposable {
    private ws: WebSocket | null = null;
    private wsUrl: string = "";
    private reconnectAttempts = 0;
    private readonly MAX_RECONNECT_DELAY_MS = 10000;
    private reconnectTimer: number | null = null; // just to track the timer
    private readonly eventBus = new EventBus<AppEventMap>();

    constructor(bus: EventBus<AppEventMap>) {
        this.eventBus = bus;
    }

    public connect(url: string) {
        if (this.reconnectTimer !== null) { // just allowing one timer to exist no parallelism
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        this.wsUrl = url;
        try {
            this.ws = new WebSocket(url);
            this.ws.binaryType = "arraybuffer";
            
            this.ws.onopen = () => {
                console.log("Connected to relay");
                this.reconnectAttempts = 0;
                this.eventBus.emit('connection:connected');
            };
            
            this.ws.onclose = () => {
                console.log("Disconnected");
                this.eventBus.emit('connection:disconnected');
                this.scheduleReconnect();
            };

            this.ws.onerror = (err) => {
                console.error("Websocket Error:", err);
                this.eventBus.emit('connection:error', err);
            };
            
            this.ws.onmessage = (e) => {
                if (typeof e.data === "string") {
                    this.eventBus.emit('network:message', e.data);
                } else if (e.data instanceof ArrayBuffer) {
                    this.eventBus.emit('network:packet', e.data);
                }
            };
        } catch (err) {
            console.error("Websocket construction failed: ", err);
            this.scheduleReconnect();
        }
    }

    public send(msg: string) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(msg);
        }
    }

    public dispose(): void {
        if (this.reconnectTimer !== null) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        if (this.ws) {
            // on purpose, handlers were nullified to avoid mem leaks and ghost events
            this.ws.onopen = null;
            this.ws.onclose = null;
            this.ws.onerror = null;
            this.ws.onmessage = null;
            this.ws.close();
            this.ws = null;
        }
    }

    private scheduleReconnect() {
        if (this.reconnectTimer !== null) {
            clearTimeout(this.reconnectTimer);
        }
        const delay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), this.MAX_RECONNECT_DELAY_MS);
        this.reconnectAttempts++;
        
        this.eventBus.emit('connection:reconnecting', delay);
        
        this.reconnectTimer = window.setTimeout(() => {
            this.reconnectTimer = null; // just to make sure that the logic syncs in connect
            this.connect(this.wsUrl);
        }, delay);
    }
}