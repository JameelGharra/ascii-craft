import { HuffmanDecoder } from './decoder/huffman';
import { decodeRLE } from './decoder/rle';
import { BinaryReader, FLAG_IS_COMPRESSED, FLAG_IS_DELTA, FLAG_METHOD, SHIFT_TABLE_MODE, MASK_TABLE_MODE, TABLE_MODE_RAW } from './protocol';
import { Renderer } from "./renderer";


const WIDTH = 212;
const HEIGHT = 66;
const TOTAL_PIXELS = WIDTH * HEIGHT;
const MAX_CHAT_MESSAGES = 35; // would delete the early ones in the queue, set to 35 to have some scroll

class GameClient {
    private ws: WebSocket | null = null;
    private renderer: Renderer;
    private huffman: HuffmanDecoder;

    private prevFrame: Uint8Array;
    private currFrame: Uint8Array;
    private  lastSeq: number; // detecting frame gaps

    private hasReceivedKeyFrame = false;

    // ui elements
    private chatLog: HTMLDivElement;
    private chatInput: HTMLInputElement;
    private statusEl: HTMLDivElement;
    

    constructor(canvasId: string) {
        const canvas = document.getElementById(canvasId) as HTMLCanvasElement;
        this.renderer = new Renderer(canvas, WIDTH, HEIGHT);
        this.prevFrame = new Uint8Array(TOTAL_PIXELS);
        this.currFrame = new Uint8Array(TOTAL_PIXELS);
        this.huffman = new HuffmanDecoder();
        this.lastSeq = -1;
        this.chatLog = document.getElementById("chat-log") as HTMLDivElement;
        this.chatInput = document.getElementById("chat-input") as HTMLInputElement;
        this.statusEl = document.getElementById("status") as HTMLDivElement;
        this.setupInputListener();
    }

    public connect(url: string) {
       this.ws = new WebSocket(url);
        this.ws.binaryType = "arraybuffer";

        this.ws.onopen = () => {
            console.log("Connected to relay");
            if(this.statusEl) this.statusEl.innerText = "Connected";
        }
        this.ws.onclose = () => {
            console.log("Disconnected");
            if(this.statusEl) this.statusEl.innerText = "Disconnected";
        }
        this.ws.onmessage = (e) => {
            if(typeof e.data === "string") {
                this.handleChatMessage(e.data);
            } else if(e.data instanceof ArrayBuffer) {
                this.handlePacket(e.data);
            }
        }
    }

    private setupInputListener() {
        this.chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                const msg = this.chatInput.value.trim();
                if (msg) {
                    this.sendCommand(msg);
                    this.chatInput.value = ''; // clear input
                }
            }
        });
    }
    
    private appendChat(sender: string, message: string, cssClass: string) {
        const el = document.createElement("div");
        el.className = "chat-msg";
        el.innerHTML = `<span class="${cssClass}">${sender}:</span> <span class="msg-text">${this.escapeHtml(message)}</span>`;
        this.chatLog.appendChild(el);
        while (this.chatLog.childElementCount > MAX_CHAT_MESSAGES) {
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
            for(let i = 0; i < TOTAL_PIXELS; i++) {
                this.currFrame[i] = this.prevFrame[i] ^ this.currFrame[i];
            }
        }
        this.renderer.render(this.currFrame);
        this.prevFrame.set(this.currFrame);
    }
}

const client = new GameClient('gameCanvas');
client.connect("ws://localhost:8080/ws");