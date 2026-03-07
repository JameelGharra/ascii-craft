export class ConnectionManager {
    private ws: WebSocket | null = null;
    private wsUrl: string = "";
    private reconnectAttempts = 0;
    private readonly MAX_RECONNECT_DELAY_MS = 10000;

    // callbacks for the orchestrator to listen to
    public onConnect?: () => void;
    public onDisconnect?: () => void;
    public onReconnectAttempt?: (delayMs: number) => void;
    public onMessage?: (msg: string) => void;
    public onPacket?: (buffer: ArrayBuffer) => void;

    public connect(url: string) {
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
        const delay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), this.MAX_RECONNECT_DELAY_MS);
        this.reconnectAttempts++;
        
        this.onReconnectAttempt?.(delay);
        
        setTimeout(() => {
            this.connect(this.wsUrl);
        }, delay);
    }
}