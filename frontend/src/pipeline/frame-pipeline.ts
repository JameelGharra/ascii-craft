import { HuffmanDecoder } from '../decoder/huffman';
import { decodeRLE } from '../decoder/rle';
import { BinaryReader, FLAG_IS_COMPRESSED, FLAG_IS_DELTA, FLAG_METHOD, SHIFT_TABLE_MODE, MASK_TABLE_MODE, TABLE_MODE_RAW } from '../protocol';
import { Renderer } from "../renderer";

export class FramePipeline {
    private renderer: Renderer;
    private huffman: HuffmanDecoder;
    private prevFrame: Uint8Array;
    private currFrame: Uint8Array;
    
    private totalPixels: number;
    private lastSeq: number = -1;
    private hasReceivedKeyFrame = false;

    constructor(canvasId: string, width: number, height: number) {
        this.totalPixels = width * height;
        const canvas = document.getElementById(canvasId) as HTMLCanvasElement;
        this.renderer = new Renderer(canvas, width, height);
        this.prevFrame = new Uint8Array(this.totalPixels);
        this.currFrame = new Uint8Array(this.totalPixels);
        this.huffman = new HuffmanDecoder();
    }

    public resetSyncState() {
        this.lastSeq = -1;
        this.hasReceivedKeyFrame = false;
    }

    public handlePacket(rawBuffer: ArrayBuffer) {
        const reader = new BinaryReader(rawBuffer);

        const flags = reader.readByte();
        const isCompressed = (flags & FLAG_IS_COMPRESSED) !== 0;
        const isHuffman = (flags & FLAG_METHOD) !== 0;
        const isDelta = (flags & FLAG_IS_DELTA) !== 0;
        const tableMode = (flags & MASK_TABLE_MODE) >> SHIFT_TABLE_MODE;

        const seq = reader.readVarint();
        const dataLen = reader.readVarint(); 
        
        console.log(`Received packet: seq=${seq}, compressed=${isCompressed}, method=${isHuffman ? "huffman" : "rle"}, delta=${isDelta}, tableMode=${tableMode}`);
        
        if (this.lastSeq !== -1) {
            if (seq === this.lastSeq) return;
            
            if (seq < this.lastSeq) {
                this.hasReceivedKeyFrame = false;
            } else if (seq > this.lastSeq + 1) {
                console.warn(`Dropped frame(s) detected! Jumped from ${this.lastSeq} to ${seq}. Waiting for I-Frame.`);
                this.hasReceivedKeyFrame = false; 
            }
        }
        
        this.lastSeq = seq;

        if (isDelta && !this.hasReceivedKeyFrame) {
            return; 
        }
        
        if (!isDelta) {
            this.hasReceivedKeyFrame = true;
        }

        if (isCompressed && isHuffman) {
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
            const payload = reader.readRemaining();
            decodeRLE(payload, this.currFrame);
        } else {
            const payload = reader.readRemaining();
            this.currFrame.set(payload);
        }

        if (isDelta) {
            for (let i = 0; i < this.totalPixels; i++) {
                this.currFrame[i] = this.prevFrame[i] ^ this.currFrame[i];
            }
        }

        this.renderer.render(this.currFrame);
        this.prevFrame.set(this.currFrame);
    }
}