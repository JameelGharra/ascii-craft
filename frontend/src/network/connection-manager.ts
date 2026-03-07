export class ConnectionManager {
    private ws: WebSocket | null = null;
    private wsUrl: string = "";
    private reconnectAttempts = 0;
    private readonly MAX_RECONNECT_DELAY_MS = 10000;
    private reconnectTimer: number | null = null; // just to track the timer

    // callbacks for the orchestrator to listen to
    public onConnect?: () => void;
    public onDisconnect?: () => void;
    public onReconnectAttempt?: (delayMs: number) => void;
    public onMessage?: (msg: string) => void;
    public onPacket?: (buffer: ArrayBuffer) => void;
    public onError?: (err: Event) => void;

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
                this.onConnect?.();
            };
            
            this.ws.onclose = () => {
                console.log("Disconnected");
                this.onDisconnect?.();
                this.scheduleReconnect();
            };

            this.ws.onerror = (err) => {
                console.error("Websocket Error:", err);
                this.onError?.(err);
            };
            
            this.ws.onmessage = (e) => {
                if (typeof e.data === "string") {
                    this.onMessage?.(e.data);
                } else if (e.data instanceof ArrayBuffer) {
                    this.onPacket?.(e.data);
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

    private scheduleReconnect() {
        if (this.reconnectTimer !== null) {
            clearTimeout(this.reconnectTimer);
        }
        const delay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), this.MAX_RECONNECT_DELAY_MS);
        this.reconnectAttempts++;
        
        this.onReconnectAttempt?.(delay);
        
        this.reconnectTimer = window.setTimeout(() => {
            this.reconnectTimer = null; // just to make sure that the logic syncs in connect
            this.connect(this.wsUrl);
        }, delay);
    }
}