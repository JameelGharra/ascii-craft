const ASCII_PALETTE = " .'`^\\\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$";
const PALETTE_COUNT = 70;

export class AsciiRenderer {
    private ctx: CanvasRenderingContext2D;
    private width: number;
    private height: number;
    private cellWidth: number = 0;
    private cellHeight: number = 0;
    private fontWidth: number = 1; // tracks natural font width
    
    // caching
    private colorLUT: string[] = new Array(256);
    private charLUT: string[] = new Array(256);
    private yPositions: number[] = [];

    constructor(canvas: HTMLCanvasElement, width: number, height: number) {
        this.width = width;
        this.height = height;

        const ctx = canvas.getContext('2d', { alpha: false });
        if (!ctx) throw new Error("Could not get 2D context for AsciiRenderer");
        this.ctx = ctx;

        // Precompute the Lookup Tables once for O(1) frame rendering
        this.precomputeLUTs();
        
        // Setup canvas size and font state immediately
        this.activate();
    }

    private precomputeLUTs() {
        for (let i = 0; i < 256; i++) {
            // Extract the 3-3-2 RGB values
            const r = Math.floor((((i >> 5) & 0x07) * 255) / 7);
            const g = Math.floor((((i >> 2) & 0x07) * 255) / 7);
            const b = Math.floor(((i & 0x03) * 255) / 3);

            // Save the exact CSS color string
            this.colorLUT[i] = `rgb(${r},${g},${b})`;

            // Calculate perceptual brightness (luminance matching the C/Go server)
            const brightness = (0.2126 * r + 0.7152 * g + 0.0722 * b);
            const paletteIndex = Math.floor((brightness / 255.0) * (PALETTE_COUNT - 1));
            
            this.charLUT[i] = ASCII_PALETTE[paletteIndex];
        }
    }

    private setupContext() {
        // 1. Set font size strictly to the cell height to fill vertical space
        const fontSize = Math.ceil(this.cellHeight); 
        this.ctx.font = `${fontSize}px 'JetBrains Mono', monospace, Courier`;
        this.ctx.textBaseline = 'top';
        this.ctx.textAlign = 'left';
        
        // 2. Measure exactly how wide the font naturally rendered
        const metrics = this.ctx.measureText('M');
        this.fontWidth = metrics.width;
    }

    public render(buffer8Bit: Uint8Array) {
        this.ctx.fillStyle = '#000000';
        this.ctx.fillRect(0, 0, this.ctx.canvas.width, this.ctx.canvas.height);

        // Save context state before applying the transform
        this.ctx.save();
        
        // MAGIC FIX: Stretch the canvas horizontally so the font's natural width 
        // perfectly matches our physical cellWidth. This eliminates ALL gaps.
        this.ctx.scale(this.cellWidth / this.fontWidth, 1);

        for (let y = 0; y < this.height; y++) {
            const yPos = this.yPositions[y];
            
            let currentStr = "";
            let currentColor = -1;
            let startX = 0;

            for (let x = 0; x < this.width; x++) {
                const idx = y * this.width + x;
                const pixel8Bit = buffer8Bit[idx];

                // Note: We removed `if (pixel8Bit === 0) continue;`
                // We WANT to draw black pixels so they act as contiguous "glue" in the string batch!

                if (pixel8Bit !== currentColor) {
                    if (currentStr !== "") {
                        this.ctx.fillStyle = this.colorLUT[currentColor];
                        // Draw at the natural fontWidth coordinate, the ctx.scale handles the expansion
                        this.ctx.fillText(currentStr, startX * this.fontWidth, yPos);
                    }
                    currentColor = pixel8Bit;
                    currentStr = this.charLUT[pixel8Bit];
                    startX = x;
                } else {
                    currentStr += this.charLUT[pixel8Bit];
                }
            }

            if (currentStr !== "") {
                this.ctx.fillStyle = this.colorLUT[currentColor];
                this.ctx.fillText(currentStr, startX * this.fontWidth, yPos);
            }
        }
        
        // Restore context state (removes the scale so the clearRect works next frame)
        this.ctx.restore();
    }

    /**
     * Called whenever ASCII mode is toggled ON.
     * Restores the high-DPI physical resolution and font states.
     */
    public activate() {
        this.ctx.canvas.width = this.ctx.canvas.clientWidth * window.devicePixelRatio;
        this.ctx.canvas.height = this.ctx.canvas.clientHeight * window.devicePixelRatio;
        
        this.cellWidth = this.ctx.canvas.width / this.width;
        this.cellHeight = this.ctx.canvas.height / this.height;

        this.yPositions = new Array(this.height);
        for(let y = 0; y < this.height; y++) {
            this.yPositions[y] = Math.floor(y * this.cellHeight);
        }
        
        this.setupContext();
    }
}