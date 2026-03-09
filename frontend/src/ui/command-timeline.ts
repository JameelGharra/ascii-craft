export class CommandTimeline {
    private container: HTMLElement;
    private maxChips = 5;

    constructor() {
        this.container = document.getElementById("command-timeline") as HTMLElement;
    }

    public addCommand(command: string, votes: number) {
        const chip = document.createElement("div");
        chip.className = "command-chip";
        chip.innerText = `${command} (${votes})`;

        // Prepend so new items appear on the left, pushing old ones right
        this.container.prepend(chip);

        while (this.container.childElementCount > this.maxChips) {
            this.container.removeChild(this.container.lastChild!);
        }
    }
}