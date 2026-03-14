import { ConnectionManager } from "../network/connection-manager";

export class LatencyTracker {
    private pingInterval: number | null = null;
    private smoothedRtt: number = 0;
    private static readonly ALPHA = 0.3; // smoothing factor from 0.0 to 1.0
    private static readonly PING_INTERVAL_MS = 3000;
    private connectionManager: ConnectionManager;

    constructor(connectionManager: ConnectionManager) {
        this.connectionManager = connectionManager;
    }

    public start() {
        this.stop();
        this.sendPing();
        this.pingInterval = window.setInterval(() => this.sendPing(), LatencyTracker.PING_INTERVAL_MS);
    }

    public stop() {
        if (this.pingInterval !== null) {
            window.clearInterval(this.pingInterval);
            this.pingInterval = null;
        }
    }

    private sendPing() {
        const payload = JSON.stringify({ type: "ping", t: performance.now() });
        this.connectionManager.send(payload);
    }

    public handlePong(originalTime: number) {
        const rtt = performance.now() - originalTime;
        if (this.smoothedRtt === 0) {
            this.smoothedRtt = rtt;
        } else {
            this.smoothedRtt = (LatencyTracker.ALPHA * rtt) + ((1 - LatencyTracker.ALPHA) * this.smoothedRtt);
        }
    }

    public getLatency(): number {
        return this.smoothedRtt;
    }
}