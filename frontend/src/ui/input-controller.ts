import { EventBus, type AppEventMap, type IDisposable } from "../core";
import type { AppConfig } from "../types";

export class InputController implements IDisposable {
    private chatInput: HTMLInputElement;
    private config: AppConfig;
    private readonly bus: EventBus<AppEventMap>;


    private standardCommandSet: Set<string>;
    private isOnCooldown = false;
    private isSystemDisabled = false; // like no feed case
    
    // command history state
    private readonly MAX_HISTORY_LENGTH = 20;
    private history: string[] =[];
    private historyIndex = -1;

    constructor(bus: EventBus<AppEventMap>, config: AppConfig) {
        this.bus = bus;
        this.config = config;
        this.standardCommandSet = new Set(config.commands.standard);
        this.chatInput = document.getElementById("chat-input") as HTMLInputElement;
        this.chatInput.addEventListener('keydown', this.handleKeyDown);
    }
    
    /**
     * Updates the allowed command sets when server configuration changes.
     */
    public updateConfig(config: AppConfig) { 
        this.config = config; 
        this.standardCommandSet = new Set(config.commands.standard); 
    }

    /**
     * Toggles the input field state based on global system connection status.
     */
    public setSystemDisabled(disabled: boolean, placeholderMessage: string = "Enter command (!w, !jump, etc)...") {
        this.isSystemDisabled = disabled;
        if (disabled) {
            this.chatInput.disabled = true;
            this.chatInput.placeholder = placeholderMessage;
        } else if (!this.isOnCooldown) {
            // only enable it if we are not currently serving a spam cooldown
            this.chatInput.disabled = false;
            this.chatInput.placeholder = placeholderMessage;
        }
    }

    public dispose(): void {
        this.chatInput.removeEventListener('keydown', this.handleKeyDown);
    }

    private handleKeyDown = (e: KeyboardEvent): void => {
        if (e.key === 'Enter') {
            if (this.isOnCooldown || this.isSystemDisabled) return;
            
            const msg = this.chatInput.value.trim().toLowerCase();
            if (msg) {
                if (this.isValidCommand(msg)) {
                    this.history.push(msg);
                    if (this.history.length > this.MAX_HISTORY_LENGTH) this.history.shift();
                    this.historyIndex = this.history.length;
                    
                    this.bus.emit('input:command_valid', msg);
                    this.triggerCooldown();
                } else {
                    this.bus.emit('input:command_invalid', msg);
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
            if (!this.isSystemDisabled) {
                this.chatInput.disabled = false;
                this.chatInput.placeholder = "Enter command (!w, !jump, etc)...";
                this.chatInput.focus();
            }
        }, this.config.chat.cooldown_ms);
    }
}