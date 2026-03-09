export interface VoteEvent {
    command: string;
    votes: number;
}

export class MessageRouter {
    public onVote?: (event: VoteEvent) => void;
    public onViewers?: (count: number) => void;
    public onPong?: (originalTime: number) => void;
    public onSystemMessage?: (msg: string) => void;

    public handleMessage(data: string) {
        if (data.startsWith('{')) {
            try {
                const parsed = JSON.parse(data);
                switch (parsed.type) {
                    case 'vote':
                        this.onVote?.({ command: parsed.command, votes: parsed.votes });
                        break;
                    case 'viewers':
                        this.onViewers?.(parsed.count);
                        break;
                    case 'pong':
                        this.onPong?.(parsed.t);
                        break;
                    default:
                        this.onSystemMessage?.(data);
                }
                return;
            } catch(err) {
                throw new Error(`Failed to parse message: ${data}`);
            }
        }
        this.onSystemMessage?.(data);
    }
}