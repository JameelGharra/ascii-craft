export interface VoteEvent {
    command: string;
    votes: number;
}

export class MessageRouter {
    public onVote?: (event: VoteEvent) => void;
    public onViewers?: (count: number) => void;
    public onPong?: (originalTime: number) => void;
    public onSystemMessage?: (msg: string) => void;
    public onReloadConfig?: () => void;

    public handleMessage(data: string) {
        if (data.startsWith('{')) {
            try {
                const parsed = JSON.parse(data);
                switch (parsed.type) {
                    case 'reload_config':
                        this.onReloadConfig?.();
                        break;
                    case 'vote':
                        if (typeof parsed.votes !== 'number' || typeof parsed.command !== 'string') {
                            throw new Error(`Invalid vote message format: ${data}`);
                        }
                        this.onVote?.({ command: parsed.command, votes: parsed.votes });
                        break;
                    case 'viewers':
                        if (typeof parsed.count !== 'number') {
                            throw new Error(`Invalid viewers message format: ${data}`);
                        }
                        this.onViewers?.(parsed.count);
                        break;
                    case 'pong':
                        if (typeof parsed.t !== 'number') {
                            throw new Error(`Invalid pong message format: ${data}`);
                        }
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