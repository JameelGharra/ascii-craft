import type { AppConfig } from "../types";

export class InputController {
    private chatInput: HTMLInputElement;
    private config: AppConfig;
    private standardCommandSet: Set<string>;
    private isOnCooldown = false;

    // Callbacks
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
                        this.onValidCommand?.(msg);
                        this.triggerCooldown();
                    } else {
                        this.onInvalidCommand?.(msg);
                    }
                    this.chatInput.value = '';
                }
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
        this.chatInput.placeholder = "Cooldown...";
        
        setTimeout(() => {
            this.isOnCooldown = false;
            this.chatInput.disabled = false;
            this.chatInput.placeholder = "Type a command and press Enter...";
            this.chatInput.focus();
        }, this.config.chat.cooldown_ms);
    }
}