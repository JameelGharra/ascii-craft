import { HuffmanDecoder } from './decoder/huffman';
import { decodeRLE } from './decoder/rle';
import { BinaryReader, FLAG_IS_COMPRESSED, FLAG_IS_DELTA, FLAG_METHOD, SHIFT_TABLE_MODE, MASK_TABLE_MODE, TABLE_MODE_RAW } from './protocol';
import { Renderer } from "./renderer";


const WIDTH = 212;
const HEIGHT = 66;
const TOTAL_PIXELS = WIDTH * HEIGHT;

class GameClient {
    private ws: WebSocket | null = null;
    private renderer: Renderer;
    private huffman: HuffmanDecoder;

    private prevFrame: Uint8Array;
    private currFrame: Uint8Array;
    private  lastSeq: number; // detecting frame gaps

    private hasReceivedKeyFrame = false;

    constructor(canvasId: string) {
        const canvas = document.getElementById(canvasId) as HTMLCanvasElement;
        this.renderer = new Renderer(canvas, WIDTH, HEIGHT);
        this.prevFrame = new Uint8Array(TOTAL_PIXELS);
        this.currFrame = new Uint8Array(TOTAL_PIXELS);
        this.huffman = new HuffmanDecoder();
        this.lastSeq = -1;
    }

    public connect(url: string) {
       this.ws = new WebSocket(url);
        this.ws.binaryType = "arraybuffer";

        const statusEl = document.getElementById("status");

        this.ws.onopen = () => {
            console.log("Connected to relay");
            if(statusEl) statusEl.innerText = "Connected";
        }
        this.ws.onclose = () => {
            console.log("Disconnected");
            if(statusEl) statusEl.innerText = "Disconnected";
        }
        this.ws.onmessage = (e) => this.handlePacket(e.data);
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