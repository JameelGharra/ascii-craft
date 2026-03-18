export class StatusOverlay {
    private readonly overlayEl: HTMLElement;

    constructor() {
        this.overlayEl = document.getElementById("game-overlay") as HTMLElement;
    }

    /**
     * Displays the initial connection handshake state.
     */
    public showConnecting() {
        this.overlayEl.classList.remove("hidden", "error-state");
        this.overlayEl.innerText = "ESTABLISHING UPLINK...";
    }

    /**
     * Displays a connection lost warning with the retry countdown.
     */
    public showReconnecting(seconds: number) {
        this.overlayEl.classList.remove("hidden");
        this.overlayEl.classList.add("error-state");
        this.overlayEl.innerText = `CONNECTION LOST\n\nRETRYING IN ${seconds}S`;
    }

    /**
     * Displays a warning when the WebSocket is open but no frames are arriving.
     */
    public showNoSignal() {
        this.overlayEl.classList.remove("hidden", "error-state");
        this.overlayEl.classList.add("warning-state");
        this.overlayEl.innerHTML = `<span class="retro-spinner">⚙</span> NO VIDEO FEED`;
    }

    /**
     * Hides the overlay, revealing the active game canvas.
     */
    public hide() {
        this.overlayEl.classList.add("hidden");
    }
}