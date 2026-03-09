import type { AppConfig } from "../types";

export class InputController {
    private chatInput: HTMLInputElement;
    private config: AppConfig;
    private standardCommandSet: Set<string>;
    private isOnCooldown = false;
    
    // command history state
    private history: string[] =[];
    private historyIndex = -1;

    // callbacks
    public onValidCommand?: (cmd: string) => void;
    public onInvalidCommand?: (cmd: string) => void;

    constructor(config: AppConfig) {
        this.config = config;
        this.standardCommandSet = new Set(config.commands.standard);
        this.chatInput = document.getElementById("chat-input") as HTMLInputElement;
        this.setupInputListener();
    }

    private setupInputListener() {
        this.chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                if (this.isOnCooldown) return;
                
                const msg = this.chatInput.value.trim().toLowerCase();
                if (msg) {
                    if (this.isValidCommand(msg)) {
                        // Push to history
                        this.history.push(msg);
                        if (this.history.length > 20) this.history.shift();
                        this.historyIndex = this.history.length;
                        
                        this.onValidCommand?.(msg);
                        this.triggerCooldown();
                    } else {
                        this.onInvalidCommand?.(msg);
                    }
                    this.chatInput.value = '';
                }
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                if (this.history.length > 0 && this.historyIndex > 0) {
                    this.historyIndex--;
                    this.chatInput.value = this.history[this.historyIndex];
                }
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                if (this.historyIndex < this.history.length - 1) {
                    this.historyIndex++;
                    this.chatInput.value = this.history[this.historyIndex];
                } else {
                    this.historyIndex = this.history.length;
                    this.chatInput.value = '';
                }
            } else if (e.key === 'Escape') {
                this.chatInput.value = '';
                this.historyIndex = this.history.length;
            }
        });
    }

    private isValidCommand(cmd: string): boolean {
        for (const [prefix, bounds] of Object.entries(this.config.commands.parameterized)) {
            if (cmd.startsWith(prefix + " ")) {
                const parts = cmd.split(" ");
                if (parts.length === 2) {
                    const val = parseInt(parts[1], 10);
                    return !isNaN(val) && val >= bounds.min && val <= bounds.max;
                }
            }
        }
        return this.standardCommandSet.has(cmd);
    }

    private triggerCooldown() {
        this.isOnCooldown = true;
        this.chatInput.disabled = true;
        this.chatInput.placeholder = "COOLDOWN...";
        
        setTimeout(() => {
            this.isOnCooldown = false;
            this.chatInput.disabled = false;
            this.chatInput.placeholder = "Enter command (!w, !jump, etc)...";
            this.chatInput.focus();
        }, this.config.chat.cooldown_ms);
    }
}