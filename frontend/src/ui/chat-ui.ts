export class ChatUI {
    private chatLog: HTMLDivElement;
    private maxMessages: number;

    constructor(maxMessages: number) {
        this.maxMessages = maxMessages;
        this.chatLog = document.getElementById("chat-log") as HTMLDivElement;
    }

    public appendChat(sender: string, message: string, cssClass: string) {
        const el = document.createElement("div");
        el.className = "chat-msg";

        // Add Timestamp
        const time = new Date().toLocaleTimeString('en-US', { hour12: false });
        const timeSpan = document.createElement("span");
        timeSpan.className = "time";
        timeSpan.textContent = `[${time}]`;

        const senderSpan = document.createElement("span");
        senderSpan.className = cssClass; 
        senderSpan.textContent = `${sender}: `;

        const msgSpan = document.createElement("span");
        msgSpan.className = "msg-text";
        msgSpan.textContent = message;

        el.appendChild(timeSpan);
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