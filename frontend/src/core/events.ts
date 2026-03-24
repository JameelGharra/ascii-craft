export interface AppEventMap {
    [key: string]: (...args: any[]) => void;
    
    // --- Connection Lifecycle ---
    'connection:connected': () => void;
    'connection:disconnected': () => void;
    'connection:reconnecting': (delayMs: number) => void;
    'connection:error': (err: Event) => void;

    // --- Raw Network Data ---
    'network:packet': (buffer: ArrayBuffer) => void;
    'network:message': (msg: string) => void;

    // --- Parsed Server Messages ---
    'server:config_reload': () => void;
    'server:vote_result': (command: string, votes: number) => void;
    'server:viewers_update': (count: number) => void;
    'server:pong': (originalTime: number) => void;
    'server:system_message': (msg: string) => void;

    // --- User Input ---
    'input:command_valid': (cmd: string) => void;
    'input:command_invalid': (cmd: string) => void;

    // --- UI Events ---
    'ui:ascii_toggle': (enabled: boolean) => void;
    'ui:chat_clear': () => void;
    'ui:chat_help': (categories: { name: string, commands: string }[]) => void;
}