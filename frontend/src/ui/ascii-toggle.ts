import { EventBus, type AppEventMap } from "../core";

export class AsciiToggle {
    private readonly bus: EventBus<AppEventMap>;
    private readonly wrapper: HTMLElement;
    private readonly checkbox: HTMLInputElement;

    constructor(bus: EventBus<AppEventMap>) {
        this.bus = bus;

        // 1. Create UI elements
        this.wrapper = document.createElement("div");
        this.wrapper.className = "ascii-toggle-wrapper";

        const label = document.createElement("span");
        label.className = "ascii-toggle-label";
        label.innerText = "ASCII MODE";

        const switchLabel = document.createElement("label");
        switchLabel.className = "ascii-toggle-switch";

        this.checkbox = document.createElement("input");
        this.checkbox.type = "checkbox";
        this.checkbox.className = "ascii-toggle-input";

        const track = document.createElement("span");
        track.className = "ascii-toggle-track";

        switchLabel.appendChild(this.checkbox);
        switchLabel.appendChild(track);

        this.wrapper.appendChild(label);
        this.wrapper.appendChild(switchLabel);

        const header = document.getElementById("status-header");
        const badge = document.getElementById("resolution-badge");
        
        if (header && badge) {
            // Group them so flexbox "space-between" keeps the status perfectly centered
            const rightGroup = document.createElement("div");
            rightGroup.className = "header-right-group";
            
            header.insertBefore(rightGroup, badge);
            rightGroup.appendChild(this.wrapper);
            rightGroup.appendChild(badge);
        }

        // 3. Bind events
        this.checkbox.addEventListener("change", this.handleChange);
    }

    private handleChange = (e: Event) => {
        const enabled = (e.target as HTMLInputElement).checked;
        this.bus.emit('ui:ascii_toggle', enabled);
    };
    
    public dispose() {
        this.checkbox.removeEventListener("change", this.handleChange);
        this.wrapper.remove();
    }
}