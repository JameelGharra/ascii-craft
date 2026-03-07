export class ChatUI {
    private chatLog: HTMLDivElement;
    private statusEl: HTMLDivElement;
    private maxMessages: number;

    constructor(maxMessages: number) {
        this.maxMessages = maxMessages;
        this.chatLog = document.getElementById("chat-log") as HTMLDivElement;
        this.statusEl = document.getElementById("status") as HTMLDivElement;
    }

    public updateStatus(text: string) {
        if (this.statusEl) {
            this.statusEl.innerText = text;
        }
    }

    public appendChat(sender: string, message: string, cssClass: string) {
        const el = document.createElement("div");
        el.className = "chat-msg";
        // note XSS vulnerability from feedback remains here for now
        el.innerHTML = `<span class="${cssClass}">${sender}:</span> <span class="msg-text">${this.escapeHtml(message)}</span>`;
        
        this.chatLog.appendChild(el);
        
        while (this.chatLog.childElementCount > this.maxMessages) {
            this.chatLog.removeChild(this.chatLog.firstChild!);
        }
        this.chatLog.scrollTop = this.chatLog.scrollHeight;
    }

    private escapeHtml(unsafe: string) {
        return unsafe
             .replace(/&/g, "&amp;")
             .replace(/</g, "&lt;")
             .replace(/>/g, "&gt;")
             .replace(/"/g, "&quot;")
             .replace(/'/g, "&#039;");
    }
}