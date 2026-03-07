export class Renderer {
    private ctx: CanvasRenderingContext2D;
    private imageData: ImageData;
    private buf32: Uint32Array; // fast pixel writing

    constructor(canvas: HTMLCanvasElement, width: number, height: number) {
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext('2d', {alpha: false});
        if(!ctx) throw new Error("Could not get 2D context");
        this.ctx = ctx;
        this.imageData = ctx.createImageData(width, height);
        this.buf32 = new Uint32Array(this.imageData.data.buffer);
    }

    public render(buffer8Bit: Uint8Array) {
        const len = buffer8Bit.length;
        
        for(let i = 0; i < len; i++) {
            const color8 = buffer8Bit[i];

            // I gave 3 bits for red, green and just 2 for blues in Go side
            const r = Math.floor((((color8 >> 5) & 0x07) * 255) / 7);
            const g = Math.floor((((color8 >> 2) & 0x07) * 255) / 7);
            const b = Math.floor(((color8 & 0x03) * 255) / 3);

            this.buf32[i] = (0xFF << 24) | (b << 16) | (g << 8) | r;
        }
        this.ctx.putImageData(this.imageData, 0, 0);
    }
}