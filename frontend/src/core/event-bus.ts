import type { IDisposable } from "./disposable";

export class EventBus<TEventMap extends Record<string, (...args: any[]) => void>> implements IDisposable {
    private listeners: {
        [K in keyof TEventMap]?: Array<TEventMap[K]>
    } = {};

    public on<K extends keyof TEventMap>(event: K, handler: TEventMap[K]): void {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event]!.push(handler);
    }

    public off<K extends keyof TEventMap>(event: K, handler: TEventMap[K]): void {
        if (!this.listeners[event]) return;

        this.listeners[event] = this.listeners[event]!.filter(
            (h) => h !== handler
        ) as TEventMap[K][];
    }

    public emit<K extends keyof TEventMap>(event: K, ...args: Parameters<TEventMap[K]>): void {
        const handlers = this.listeners[event];
        if (!handlers) return;
        
        // made a copy just if handler deletes itself
        for (const handler of [...handlers]) {
            handler(...args);
        }
    }

    public dispose(): void {
        this.listeners = {};
    }
}