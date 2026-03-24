const ASCII_PALETTE = " .'`^\\\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$";
const PALETTE_COUNT = 70;

export class AsciiRenderer {
    private ctx: CanvasRenderingContext2D;
    private width: number;
    private height: number;
    private cellWidth: number = 0;
    private cellHeight: number = 0;
    
    // caching
    private colorLUT: string[] = new Array(256);
    private charLUT: string[] = new Array(256);
    private xPositions: number[] = [];
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
        // Using a monospace font so the grid aligns perfectly.
        // We use cellHeight to fill the vertical space. 
        // A multiplier of ~1.1 to 1.2 sometimes helps monospace fonts overlap slightly for a denser "image" look
        const fontSize = Math.floor(this.cellHeight * 1.1); 
        this.ctx.font = `${fontSize}px 'JetBrains Mono', monospace, Courier`;
        this.ctx.textBaseline = 'top';
        this.ctx.textAlign = 'left';
    }

    public render(buffer8Bit: Uint8Array) {
        // 1. Clear the frame (black background)
        this.ctx.fillStyle = '#000000';
        this.ctx.fillRect(0, 0, this.ctx.canvas.width, this.ctx.canvas.height);

        let lastColor = -1;

        // 2. Iterate pixels and draw characters
        for (let y = 0; y < this.height; y++) {
            const yPos = this.yPositions[y];
            for (let x = 0; x < this.width; x++) {
                const idx = y * this.width + x;
                const pixel8Bit = buffer8Bit[idx];

                // Performance optimization: don't draw pure black/empty space
                if (pixel8Bit === 0) continue;
                
                // only touch canvas api if the color actually changes
                if (pixel8Bit !== lastColor) {
                    this.ctx.fillStyle = this.colorLUT[pixel8Bit];
                    lastColor = pixel8Bit;
                }

                // Draw the precomputed ASCII character at the cell's physical coordinate
                this.ctx.fillText(this.charLUT[pixel8Bit], this.xPositions[x], yPos);
            }
        }
    }

    /**
     * Called whenever ASCII mode is toggled ON.
     * Restores the high-DPI physical resolution and font states.
     */
    public activate() {
        // Switch canvas to high-DPI physical resolution for crisp text
        this.ctx.canvas.width = this.ctx.canvas.clientWidth * window.devicePixelRatio;
        this.ctx.canvas.height = this.ctx.canvas.clientHeight * window.devicePixelRatio;
        
        this.cellWidth = this.ctx.canvas.width / this.width;
        this.cellHeight = this.ctx.canvas.height / this.height;

        this.xPositions = new Array(this.width);
        for(let x = 0; x < this.width; x++) {
            this.xPositions[x] = Math.floor(x * this.cellWidth);
        }

        this.yPositions = new Array(this.height);
        for(let y = 0; y < this.height; y++) {
            this.yPositions[y] = Math.floor(y * this.cellHeight);
        }
        
        this.setupContext();
    }
}