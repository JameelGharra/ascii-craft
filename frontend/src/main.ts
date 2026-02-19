import { decodeRLE } from './decoder/rle';
import { BinaryReader, FLAG_IS_COMPRESSED, FLAG_IS_DELTA, FLAG_METHOD } from './protocol';
import { Renderer } from "./renderer";


const WIDTH = 212;
const HEIGHT = 66;
const TOTAL_PIXELS = WIDTH * HEIGHT;

class GameClient {
    private ws: WebSocket | null = null;
    private renderer: Renderer;

    private prevFrame: Uint8Array;
    private currFrame: Uint8Array;

    private hasReceivedKeyFrame = false;

    constructor(canvasId: string) {
        const canvas = document.getElementById(canvasId) as HTMLCanvasElement;
        this.renderer = new Renderer(canvas, WIDTH, HEIGHT);
        this.prevFrame = new Uint8Array(TOTAL_PIXELS);
        this.currFrame = new Uint8Array(TOTAL_PIXELS);
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

        const seq = reader.readVarint();
        const dataLen = reader.readVarint();

        if(isDelta && !this.hasReceivedKeyFrame) {
            return ; // no reference yet gotta wait for gameserver to send i-frame
        }
        if(!isDelta) {
            this.hasReceivedKeyFrame = true;
        }
        if(isCompressed && isHuffman) {
            console.warn("Huffman not implemented on frontend yet")
            return ;
        }
        const payload = reader.readRemaining();
        if(!isCompressed) {
            this.currFrame.set(payload);
        } else {
            decodeRLE(payload, this.currFrame);
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