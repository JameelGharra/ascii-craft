import type { MetricsSnapshot } from "../metrics/metrics-collector";

export class TelemetryPanel {
    private readonly contentEl: HTMLElement;
    private lastUpdate: number = 0;

    constructor() {
        this.contentEl = document.getElementById("telemetry-content") as HTMLElement;
    }

    private formatRow(label: string, value: string, color: string): string {
        const paddedLabel = (label + " ").padEnd(18, '.');
        return `${paddedLabel} <span style="color: ${color}; font-weight: bold">${value}</span>`;
    }

    /**
     * Updates the telemetry readout with the latest statistics. 
     * Output is throttled to ~4 FPS to maintain human readability.
     */
    public update(stats: MetricsSnapshot, latencyMs: number) {
        // throttle  DOM updates to roughly 4 fps for readability
        const now = performance.now();
        if (now - this.lastUpdate < 250) return;
        this.lastUpdate = now;

        const fpsColor = stats.fps > 30 ? "var(--primary)" : (stats.fps > 15 ? "var(--warning)" : "var(--error)");
        const bwColor = stats.bandwidthKbps > 500 ? "var(--error)" : (stats.bandwidthKbps > 100 ? "var(--warning)" : "var(--primary)");
        const compColor = stats.compressionRatio > 80 ? "var(--primary)" : (stats.compressionRatio > 50 ? "var(--warning)" : "var(--error)");
        const pingColor = latencyMs > 150 ? "var(--error)" : (latencyMs > 80 ? "var(--warning)" : "var(--primary)");
        
        const frameType = stats.isKeyFrame ? "I-FRAME (FULL)" : "P-FRAME (Δ)";
        const frameColor = stats.isKeyFrame ? "var(--warning)" : "var(--primary)";

        const html =[
            this.formatRow("FPS", stats.fps.toString(), fpsColor),
            this.formatRow("BANDWIDTH", `${stats.bandwidthKbps.toFixed(2)} KB/s`, bwColor),
            this.formatRow("COMPRESSION", `${stats.compressionRatio.toFixed(1)}%`, compColor),
            this.formatRow("LATENCY", `${latencyMs.toFixed(1)}ms`, pingColor),
            this.formatRow("VIEWERS", stats.viewers.toString(), "var(--accent)"),
            this.formatRow("FRAME", frameType, frameColor),
            this.formatRow("CODEC", stats.encodingMethod, "var(--accent)")
        ].join("\n");

        this.contentEl.innerHTML = html;
        this.contentEl.classList.remove("awaiting-state"); // just for properly showing awaiting
    }
}