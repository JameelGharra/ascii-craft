import { HuffmanDecoder } from '../decoder/huffman';
import { decodeRLE } from '../decoder/rle';
import type { MetricsCollector } from '../metrics/metrics-collector';
import { BinaryReader, FLAG_IS_COMPRESSED, FLAG_IS_DELTA, FLAG_METHOD, SHIFT_TABLE_MODE, MASK_TABLE_MODE, TABLE_MODE_RAW } from '../protocol';
import { Renderer } from "../renderer";
import { AsciiRenderer } from "../ascii-renderer";

export class FramePipeline {
    private readonly huffman: HuffmanDecoder;
    private readonly metrics: MetricsCollector;
    private readonly prevFrame: Uint8Array;
    private readonly currFrame: Uint8Array;
    
    private readonly renderer: Renderer;
    private asciiRenderer: AsciiRenderer | null = null;
    private asciiMode: boolean = false;
    private readonly canvas: HTMLCanvasElement;
    private readonly width: number;
    private readonly height: number;

    private totalPixels: number;
    private lastSeq: number = -1;
    private hasReceivedKeyFrame = false;
    private pendingRender: boolean = false;
    private animationFrameId: number | null = null;


    private setupRenderingLoop() {
        const renderLoop = () => {
            if (this.pendingRender) {
                if (this.asciiMode && this.asciiRenderer) {
                    this.asciiRenderer.render(this.currFrame);
                } else {
                    this.renderer.render(this.currFrame);
                }
                this.pendingRender = false;
            }
            this.animationFrameId = requestAnimationFrame(renderLoop);
        };
        this.animationFrameId = requestAnimationFrame(renderLoop);
    }

    constructor(canvasId: string, width: number, height: number, metrics: MetricsCollector) {
        this.totalPixels = width * height;
        this.width = width;
        this.height = height;
        this.canvas = document.getElementById(canvasId) as HTMLCanvasElement;

        this.renderer = new Renderer(this.canvas, this.width, this.height);
        this.metrics = metrics;
        this.prevFrame = new Uint8Array(this.totalPixels);
        this.currFrame = new Uint8Array(this.totalPixels);
        this.huffman = new HuffmanDecoder();
        this.setupRenderingLoop();
    }

    public resetSyncState() {
        this.lastSeq = -1;
        this.hasReceivedKeyFrame = false;
    }
    
    public handlePacket(rawBuffer: ArrayBuffer) {
        try {
            const reader = new BinaryReader(rawBuffer);

            const flags = reader.readByte();
            const isCompressed = (flags & FLAG_IS_COMPRESSED) !== 0;
            const isHuffman = (flags & FLAG_METHOD) !== 0;
            const isDelta = (flags & FLAG_IS_DELTA) !== 0;
            const tableMode = (flags & MASK_TABLE_MODE) >> SHIFT_TABLE_MODE;

            const seq = reader.readVarint();
            const dataLen = reader.readVarint(); 
            
            // console.log(`Received packet: seq=${seq}, compressed=${isCompressed}, method=${isHuffman ? "huffman" : "rle"}, delta=${isDelta}, tableMode=${tableMode}`);
            
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
            }

            const payload = reader.readRemaining();
            if (payload.length !== dataLen) {
                throw new Error(`Packet corruption: declared dataLen was ${dataLen} but payload has ${payload.length} bytes.`);
            }
                

            if (isCompressed && isHuffman) {
                this.huffman.decodeStream(payload, this.currFrame);
            } else if (isCompressed) { 
                decodeRLE(payload, this.currFrame);
            } else {
                this.currFrame.set(payload);
            }

            if (isDelta) {
                for (let i = 0; i < this.totalPixels; i++) {
                    this.currFrame[i] = this.prevFrame[i] ^ this.currFrame[i];
                }
            }

            this.pendingRender = true;
            this.prevFrame.set(this.currFrame);
            let methodStr = "RAW";
            if (isCompressed) {
                methodStr = isHuffman ? "HUFFMAN" : "RLE";
            }
            this.metrics.recordFrame(rawBuffer.byteLength, !isDelta, methodStr);
        } catch(err) {
            console.error("Frame pipeline encountered a decoding error:", err);
            this.hasReceivedKeyFrame = false;
        }
    }
    public setAsciiMode(enabled: boolean) {
        this.asciiMode = enabled;
        if (enabled) {
            if (!this.asciiRenderer) {
                // Lazy instantiation
                this.asciiRenderer = new AsciiRenderer(this.canvas, this.width, this.height);
            } else {
                // Re-apply physical pixel dimensions and font states
                this.asciiRenderer.activate();
            }
        } else {
            // Restore original 1:1 pixel rendering dimensions for standard Renderer
            this.canvas.width = this.width;
            this.canvas.height = this.height;
        }
        
        // Force an immediate re-render of the current frame
        this.pendingRender = true;
    }
    public dispose(): void {
        if (this.animationFrameId !== null) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }
    }
}