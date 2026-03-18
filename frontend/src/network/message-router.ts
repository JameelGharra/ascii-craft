import { EventBus, type AppEventMap } from "../core";

export interface VoteEvent {
    command: string;
    votes: number;
}

type ServerMessage =
    | { type: 'reload_config' }
    | { type: 'vote', command: string, votes: number }
    | { type: 'viewers', count: number }
    | { type: 'pong', t: number };

export class MessageRouter {

    private readonly bus: EventBus<AppEventMap>;

    constructor(bus: EventBus<AppEventMap>) {
        this.bus = bus;
    }

    public handleMessage(data: string) {
        if (data.startsWith('{')) {
            try {
                const parsed = JSON.parse(data) as ServerMessage;
                switch (parsed.type) {
                    case 'reload_config':
                        this.bus.emit('server:config_reload');
                        break;
                    case 'vote':
                        if (typeof parsed.votes !== 'number' || typeof parsed.command !== 'string') {
                            throw new Error(`Invalid vote message format: ${data}`);
                        }
                        this.bus.emit('server:vote_result', parsed.command, parsed.votes);
                        break;
                    case 'viewers':
                        if (typeof parsed.count !== 'number') {
                            throw new Error(`Invalid viewers message format: ${data}`);
                        }
                        this.bus.emit('server:viewers_update', parsed.count);
                        break;
                    case 'pong':
                        if (typeof parsed.t !== 'number') {
                            throw new Error(`Invalid pong message format: ${data}`);
                        }
                        this.bus.emit('server:pong', parsed.t);
                        break;
                    default:
                        this.bus.emit('server:system_message', data);
                }
                return;
            } catch(err) {
                throw new Error(`Failed to parse message: ${data}`);
            }
        }
        this.bus.emit('server:system_message', data);
    }
}