export class StatusHeader {
    private statusEl: HTMLElement;
    private resolutionEl: HTMLElement;

    constructor() {
        this.statusEl = document.getElementById("status") as HTMLElement;
        this.resolutionEl = document.getElementById("resolution-badge") as HTMLElement;
    }

    public setConfig(width: number, height: number) {
        this.resolutionEl.innerText = `${width}x${height} @ 8-BIT`;
    }

    public setConnected() {
        this.statusEl.innerHTML = `<span class="status-dot status-live"></span>LIVE`;
        this.statusEl.style.color = "var(--primary)";
    }

    public setDisconnected() {
        this.statusEl.innerHTML = `<span class="status-dot status-offline"></span>OFFLINE`;
        this.statusEl.style.color = "var(--error)";
    }

    public setReconnecting(seconds: number) {
        this.statusEl.innerHTML = `<span class="status-dot status-wait"></span>RECONNECTING IN ${seconds}S`;
        this.statusEl.style.color = "var(--warning)";
    }
}