export class CommandTimeline {
    private readonly container: HTMLElement;
    private readonly MAX_CHIPS = 5;

    constructor() {
        this.container = document.getElementById("command-timeline") as HTMLElement;
    }

    /**
     * Pushes a new aggregated command result to the UI timeline.
     * Older commands are pushed to the right and automatically removed.
     */
    public addCommand(command: string, votes: number) {
        const chip = document.createElement("div");
        chip.className = "command-chip";
        chip.innerText = `${command} (${votes})`;

        this.container.prepend(chip);

        while (this.container.childElementCount > this.MAX_CHIPS) {
            this.container.removeChild(this.container.lastChild!);
        }
    }
}