export class StatusOverlay {
    private overlayEl: HTMLElement;

    constructor() {
        this.overlayEl = document.getElementById("game-overlay") as HTMLElement;
    }

    public showConnecting() {
        this.overlayEl.classList.remove("hidden", "error-state");
        this.overlayEl.innerText = "ESTABLISHING UPLINK...";
    }

    public showReconnecting(seconds: number) {
        this.overlayEl.classList.remove("hidden");
        this.overlayEl.classList.add("error-state");
        this.overlayEl.innerText = `CONNECTION LOST\n\nRETRYING IN ${seconds}S`;
    }

    public showNoSignal() {
        this.overlayEl.classList.remove("hidden", "error-state");
        this.overlayEl.classList.add("warning-state");
        this.overlayEl.innerHTML = `<span class="retro-spinner">⚙</span> NO VIDEO FEED`;
    }

    public hide() {
        this.overlayEl.classList.add("hidden");
    }
}