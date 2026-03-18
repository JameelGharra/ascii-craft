export class StatusHeader {
    private readonly statusEl: HTMLElement;
    private readonly resolutionEl: HTMLElement;

    constructor() {
        this.statusEl = document.getElementById("status") as HTMLElement;
        this.resolutionEl = document.getElementById("resolution-badge") as HTMLElement;
    }

    /**
     * Updates the resolution badge text.
     */
    public setConfig(width: number, height: number) {
        this.resolutionEl.innerText = `${width}x${height} @ 8-BIT`;
    }

    /**
     * Sets the status indicator to LIVE.
     */
    public setConnected() {
        this.statusEl.innerHTML = `<span class="status-dot status-live"></span>LIVE`;
        this.statusEl.style.color = "var(--primary)";
    }

    /**
     * Sets the status indicator to OFFLINE.
     */
    public setDisconnected() {
        this.statusEl.innerHTML = `<span class="status-dot status-offline"></span>OFFLINE`;
        this.statusEl.style.color = "var(--error)";
    }

    /**
     * Sets the status indicator to RECONNECTING.
     */
    public setReconnecting(seconds: number) {
        this.statusEl.innerHTML = `<span class="status-dot status-wait"></span>RECONNECTING IN ${seconds}S`;
        this.statusEl.style.color = "var(--warning)";
    }
}