import { HuffmanDecoder } from './decoder/huffman';
import { decodeRLE } from './decoder/rle';
import { BinaryReader, FLAG_IS_COMPRESSED, FLAG_IS_DELTA, FLAG_METHOD, SHIFT_TABLE_MODE, MASK_TABLE_MODE, TABLE_MODE_RAW } from './protocol';
import { Renderer } from "./renderer";


interface AppConfig {
    video: {
        width: number;
        height: number;
    };
    chat: {
        cooldown_ms: number; // cooldown to prevent spamming cmds
        max_messages: number; // deletes early ones in the queue, I set usually to 35 to have some scroll sense
    };
    commands: {
        standard: string[];
        parameterized: {
            [key: string]: { min: number; max: number };
        };
    };
}

function isValidAppConfig(data: any): data is AppConfig {
    return data
        && data.video && typeof data.video.width === 'number'
        && data.chat && typeof data.chat.cooldown_ms === 'number'
        && data.commands && Array.isArray(data.commands.standard);
}

class GameClient {
    private ws: WebSocket | null = null;
    private renderer: Renderer;
    private huffman: HuffmanDecoder;

    private prevFrame: Uint8Array;
    private currFrame: Uint8Array;
    private  lastSeq: number; // detecting frame gaps

    private hasReceivedKeyFrame = false;
    private isOnCooldown = false; // to prevent spamming cmds

    // ui elements
    private chatLog: HTMLDivElement;
    private chatInput: HTMLInputElement;
    private statusEl: HTMLDivElement;

    // config injected from server
    private config: AppConfig;
    private totalPixels: number;
    private standardCommandSet: Set<string>;

    // reconnect state for ws
    private wsUrl: string = "";
    private reconnectAttempts = 0;
    private readonly MAX_RECONNECT_DELAY_MS = 10000;
    

    constructor(config: AppConfig, canvasId: string) {
        this.config = config
        this.totalPixels = config.video.width * config.video.height;
        this.standardCommandSet = new Set(config.commands.standard);
        const canvas = document.getElementById(canvasId) as HTMLCanvasElement;
        this.renderer = new Renderer(canvas, this.config.video.width, this.config.video.height);
        this.prevFrame = new Uint8Array(this.totalPixels);
        this.currFrame = new Uint8Array(this.totalPixels);
        this.huffman = new HuffmanDecoder();
        this.lastSeq = -1;
        this.chatLog = document.getElementById("chat-log") as HTMLDivElement;
        this.chatInput = document.getElementById("chat-input") as HTMLInputElement;
        this.statusEl = document.getElementById("status") as HTMLDivElement;
        this.setupInputListener();
    }

    public connect(url: string) {
        this.wsUrl = url;
        try {
            this.ws = new WebSocket(url);
            this.ws.binaryType = "arraybuffer";
            this.ws.onopen = () => {
                console.log("Connected to relay");
                if(this.statusEl) this.statusEl.innerText = "Connected";
                this.reconnectAttempts = 0;
                this.lastSeq = -1;
                this.hasReceivedKeyFrame = false;
            }
            this.ws.onclose = () => {
                console.log("Disconnected");
                if(this.statusEl) this.statusEl.innerText = "Disconnected";
                this.scheduleReconnect();
            }
            this.ws.onmessage = (e) => {
                if(typeof e.data === "string") {
                    this.handleChatMessage(e.data);
                } else if(e.data instanceof ArrayBuffer) {
                    this.handlePacket(e.data);
                }
            }
        } catch(err) {
            console.error("Websocket construction failed: ", err);
            this.scheduleReconnect();
        }
    }

    // for ws, having delay with 1.5 as multipler and 10 secs cap
    private scheduleReconnect() {
        const delay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), this.MAX_RECONNECT_DELAY_MS);
        this.reconnectAttempts++;
        if(this.statusEl) {
            this.statusEl.innerText = `Reconnecting in ${Math.round(delay/1000)}s...`;
        }
        this.appendChat("System", `Connection lost. Retrying in ${Math.round(delay/1000)}s...`, "system");
        setTimeout(() => {
            this.connect(this.wsUrl);
        }, delay);
    }

    private isValidCommand(cmd: string): boolean {
        for (const [prefix, bounds] of Object.entries(this.config.commands.parameterized)) {
            if(cmd.startsWith(prefix + " ")) {
                const parts = cmd.split(" ");
                if(parts.length === 2) {
                    const val = parseInt(parts[1], 10);
                    return !isNaN(val) && val >= bounds.min && val <= bounds.max;
                }
            }
        }
        return this.standardCommandSet.has(cmd);
    }

    private setupInputListener() {
        this.chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                if(this.isOnCooldown) {
                    return;
                }
                const msg = this.chatInput.value.trim().toLowerCase();
                if (msg) {
                    if(this.isValidCommand(msg)) {
                        this.sendCommand(msg);
                        this.triggerCooldown();
                    } else {
                        this.appendChat("System", `Invalid command: ${msg}`, "system");
                    }
                    this.chatInput.value = ''; // clear input
                }
            }
        });
    }
    
    private triggerCooldown() {
        this.isOnCooldown = true;
        this.chatInput.disabled = true;
        this.chatInput.placeholder = "Cooldown...";
        setTimeout(() => {
            this.isOnCooldown = false;
            this.chatInput.disabled = false;
            this.chatInput.placeholder = "Type command and press Enter...";
            this.chatInput.focus();
        }, this.config.chat.cooldown_ms);
    }

    private appendChat(sender: string, message: string, cssClass: string) {
        const el = document.createElement("div");
        el.className = "chat-msg";
        el.innerHTML = `<span class="${cssClass}">${sender}:</span> <span class="msg-text">${this.escapeHtml(message)}</span>`;
        this.chatLog.appendChild(el);
        while (this.chatLog.childElementCount > this.config.chat.max_messages) {
            this.chatLog.removeChild(this.chatLog.firstChild!);
        }
        this.chatLog.scrollTop = this.chatLog.scrollHeight; // auto scroll
    }

    private escapeHtml(unsafe: string) {
        return unsafe
             .replace(/&/g, "&amp;")
             .replace(/</g, "&lt;")
             .replace(/>/g, "&gt;")
             .replace(/"/g, "&quot;")
             .replace(/'/g, "&#039;");
    }
    
    private sendCommand(msg: string) {
        if(this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(msg);
            this.appendChat("You", msg, "user");
        }
    }
    private handleChatMessage(msg: string) {
        // just to tell what majority was chosen
        this.appendChat("Server", msg, "server");
    }

    private handlePacket(rawBuffer: ArrayBuffer) {
        const reader = new BinaryReader(rawBuffer);


        const flags = reader.readByte();
        const isCompressed = (flags & FLAG_IS_COMPRESSED) !== 0;
        const isHuffman = (flags & FLAG_METHOD) !== 0;
        const isDelta = (flags & FLAG_IS_DELTA) !== 0;
        const tableMode = (flags & MASK_TABLE_MODE) >> SHIFT_TABLE_MODE;


        const seq = reader.readVarint();
        const dataLen = reader.readVarint(); // have no usage for it atm, since it is at the end anyway
        console.log(`Received packet: seq=${seq}, compressed=${isCompressed}, method=${isHuffman ? "huffman" : "rle"}, delta=${isDelta}, tableMode=${tableMode}`);
        if(this.lastSeq !== -1 && seq <= this.lastSeq) {
            return ; // older packet sent (pretty rare, but why not)
        }
        if(this.lastSeq !== -1 && seq > this.lastSeq + 1) {
            this.hasReceivedKeyFrame = false;
        }
        this.lastSeq = seq;

        if(isDelta && !this.hasReceivedKeyFrame) {
            return ; // no reference yet gotta wait for gameserver to send i-frame
        }
        if(!isDelta) {
            this.hasReceivedKeyFrame = true;
        }
        if (isCompressed && isHuffman) {
            // if the table mode is in raw then there is no length prefix
            let tableData: Uint8Array;
            
            if (tableMode === TABLE_MODE_RAW) {
                tableData = reader.readSlice(256);
            } else {
                const tableLen = reader.readByte();
                tableData = reader.readSlice(tableLen);
            }

            this.huffman.decodeTable(tableMode, tableData);

            const payload = reader.readRemaining();
            this.huffman.decodeStream(payload, this.currFrame);

        } else if (isCompressed) { 
            // RLE
            const payload = reader.readRemaining();
            decodeRLE(payload, this.currFrame);
        } else {
            // raw
            const payload = reader.readRemaining();
            this.currFrame.set(payload);
        }
        if(isDelta) {
            for(let i = 0; i < this.totalPixels; i++) {
                this.currFrame[i] = this.prevFrame[i] ^ this.currFrame[i];
            }
        }
        this.renderer.render(this.currFrame);
        this.prevFrame.set(this.currFrame);
    }
}

async function fetchConfigWithRetry(url: string, maxRetries: number = 10, delayMs: number = 2000): Promise<AppConfig> {
    const statusEl = document.getElementById("status");
    for(let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
            const response = await fetch(url);
            if(!response.ok) throw new Error(`HTTP error status: ${response.status}`);
            const data = await response.json();
            if(!isValidAppConfig(data)) {
                throw new Error("Invalid config format from server");
            }
            if(statusEl) statusEl.innerText = "Config loaded, connecting...";
            return data;
        } catch(err) {
            console.warn(`Config fetch attempt ${attempt} failed:`, err);
            if(statusEl) statusEl.innerText = `Waiting for server... (Attempt ${attempt}/${maxRetries})`;
            if(attempt < maxRetries) {
                await new Promise(resolve => setTimeout(resolve, delayMs));
            }
        }
    }
    throw new Error("Could not reach to server to fetch configuration.");
}

async function initApp() {
    try {
        const config = await fetchConfigWithRetry('http://localhost:8080/api/config');
        const client = new GameClient(config, 'gameCanvas');
        client.connect("ws://localhost:8080/ws");
    } catch(err) {
        console.error("Initialization failed:", err);
        const statusEl = document.getElementById("status");
        if(statusEl) statusEl.innerText = "Failed to initialize game. Please refresh later.";
    }
}

initApp();