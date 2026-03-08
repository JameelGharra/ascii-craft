export class Renderer {
    private ctx: CanvasRenderingContext2D;
    private imageData: ImageData;
    private buf32: Uint32Array; // fast pixel writing
    private lut: Uint32Array; // lookup table (since it is only 256 colors rather precompute)

    constructor(canvas: HTMLCanvasElement, width: number, height: number) {
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext('2d', {alpha: false});
        if(!ctx) throw new Error("Could not get 2D context");
        this.ctx = ctx;
        this.imageData = ctx.createImageData(width, height);
        this.buf32 = new Uint32Array(this.imageData.data.buffer);
        this.lut = new Uint32Array(256);
        for (let i = 0; i < 256; i++) {
            // I gave 3 bits for red, green and just 2 for blues in Go side
            const r = Math.floor((((i >> 5) & 0x07) * 255) / 7);
            const g = Math.floor((((i >> 2) & 0x07) * 255) / 7);
            const b = Math.floor(((i & 0x03) * 255) / 3);

            // little endian memory layout requires ABGR order
            this.lut[i] = (0xFF << 24) | (b << 16) | (g << 8) | r;
        }
    }

    public render(buffer8Bit: Uint8Array) {
        const len = buffer8Bit.length;
        
        for(let i = 0; i < len; i++) {
            this.buf32[i] = this.lut[buffer8Bit[i]];
        }
        this.ctx.putImageData(this.imageData, 0, 0);
    }
}