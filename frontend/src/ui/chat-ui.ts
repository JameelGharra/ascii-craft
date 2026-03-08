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

        const senderSpan = document.createElement("span");
        senderSpan.className = cssClass; 
        senderSpan.textContent = `${sender}: `;

        const msgSpan = document.createElement("span");
        msgSpan.className = "msg-text";
        msgSpan.textContent = message;

        el.appendChild(senderSpan);
        el.appendChild(msgSpan);
        
        this.chatLog.appendChild(el);
        
        while (this.chatLog.childElementCount > this.maxMessages) {
            this.chatLog.removeChild(this.chatLog.firstChild!);
        }
        
        // bottom auto-scroll 
        this.chatLog.scrollTop = this.chatLog.scrollHeight;
    }
}