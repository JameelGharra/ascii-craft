export class ChatUI {
    private readonly chatLog: HTMLDivElement;
    private readonly maxMessages: number;

    constructor(maxMessages: number) {
        this.maxMessages = maxMessages;
        this.chatLog = document.getElementById("chat-log") as HTMLDivElement;
    }

    /**
     * Appends a new chat message to the log and auto-scrolls to the bottom.
     * @param sender The name of the sender (e.g., 'System', 'You')
     * @param message The message content
     * @param cssClass The CSS class to apply to the sender for styling
     */
    public appendChat(sender: string, message: string, cssClass: string) {
        const el = document.createElement("div");
        el.className = "chat-msg";

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

    /**
     * Clears all messages from the chat log.
     */
    public clearChat() {
        this.chatLog.innerHTML = '';
    }

    /**
     * Renders a structured, multi-line help message.
     */
    public appendHelp(categories: { name: string, commands: string }[]) {
        const el = document.createElement("div");
        el.className = "chat-msg";

        const time = new Date().toLocaleTimeString('en-US', { hour12: false });
        const timeSpan = document.createElement("span");
        timeSpan.className = "time";
        timeSpan.textContent = `[${time}]`;

        const senderSpan = document.createElement("span");
        senderSpan.className = "system"; 
        senderSpan.textContent = `System: `;

        const msgSpan = document.createElement("span");
        msgSpan.className = "msg-text";
        
        const title = document.createElement("div");
        title.textContent = "AVAILABLE COMMANDS:";
        title.style.color = "var(--primary)";
        title.style.fontWeight = "bold";
        title.style.marginTop = "4px";
        title.style.marginBottom = "4px";
        msgSpan.appendChild(title);

        categories.forEach(cat => {
            const catLine = document.createElement("div");
            catLine.style.marginLeft = "12px";
            catLine.style.marginBottom = "4px";
            catLine.style.lineHeight = "1.4";
            
            const catName = document.createElement("span");
            catName.textContent = `[${cat.name}] `;
            catName.style.color = "var(--warning)"; // Colors the category name gold/yellow
            catName.style.fontWeight = "bold";
            
            const catCmds = document.createElement("span");
            catCmds.textContent = cat.commands;
            catCmds.style.color = "#e0e0e0";

            catLine.appendChild(catName);
            catLine.appendChild(catCmds);
            msgSpan.appendChild(catLine);
        });

        el.appendChild(timeSpan);
        el.appendChild(senderSpan);
        el.appendChild(msgSpan);
        
        this.chatLog.appendChild(el);
        
        while (this.chatLog.childElementCount > this.maxMessages) {
            this.chatLog.removeChild(this.chatLog.firstChild!);
        }
        
        this.chatLog.scrollTop = this.chatLog.scrollHeight;
    }
}