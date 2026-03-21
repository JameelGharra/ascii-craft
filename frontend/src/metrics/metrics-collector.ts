export interface MetricsSnapshot {
    fps: number;
    bandwidthKbps: number;
    compressionRatio: number;
    isKeyFrame: boolean;
    encodingMethod: string;
    viewers: number;
}

export class MetricsCollector {
    private static readonly PRUNE_FRAME_INTERVAL_MS = 1000; // intervals for fps calc
    private readonly MAX_FRAMES = 500; // more than enough for 1 sec of 144hz video

    
    private rawFrameSizeBytes: number;
    private readonly frameTimestamps = new Float64Array(this.MAX_FRAMES);
    private readonly frameBytes = new Uint32Array(this.MAX_FRAMES);

    private head = 0;
    private tail = 0;
    private count = 0;

    private lastCompressionRatio: number = 0;
    private lastEncodingMethod: string = "RAW";
    private latchedKeyFrame: boolean = false; // if we get an i-frame display it first then vanish
    private viewersCount: number = 0;

    constructor(videoWidth: number, videoHeight: number) {
        // this assumes that 1 pixel = 1 byte btw
        this.rawFrameSizeBytes = videoWidth * videoHeight;
    }

    /**
     * Records a single frame's telemetry data into the ring buffer.
     * Executes in O(1) time with zero memory allocations.
     */
    public recordFrame(packetSizeBytes: number, isKeyFrame: boolean, encodingMethod: string) {
        const now = performance.now();
        if (this.count === this.MAX_FRAMES) {
            // just making space and dropping oldest
            this.tail = (this.tail + 1) % this.MAX_FRAMES;
            this.count--;
        }
        this.frameTimestamps[this.head] = now;
        this.frameBytes[this.head] = packetSizeBytes;
        this.head = (this.head + 1) % this.MAX_FRAMES;
        this.count++;
        this.lastCompressionRatio = Math.max(0, 1 - (packetSizeBytes / this.rawFrameSizeBytes));
        if (isKeyFrame) {
            this.latchedKeyFrame = true;
        }
        this.lastEncodingMethod = encodingMethod;
    }

    public setViewersCount(count: number) {
        this.viewersCount = count;
    }

    /**
     * Prunes old frames from the buffer and calculates the current aggregate statistics.
     * Pruning is amortized O(1).
     */
    public getSnapshot(): MetricsSnapshot {

        const now = performance.now();
        const cutoff = now - MetricsCollector.PRUNE_FRAME_INTERVAL_MS;

        while (this.count > 0 && this.frameTimestamps[this.tail] < cutoff) {
            this.tail = (this.tail + 1) % this.MAX_FRAMES;
            this.count--;
        }


        let totalBytes = 0;
        let curr = this.tail;
        for (let i = 0; i < this.count; i++) {
            totalBytes += this.frameBytes[curr];
            curr = (curr + 1) % this.MAX_FRAMES;
        }
        const fps = this.count;
        const bandwidthKbps = totalBytes / 1024; // kb/s
        const wasKeyFrame = this.latchedKeyFrame;
        this.latchedKeyFrame = false;
        return {
            fps,
            bandwidthKbps,
            compressionRatio: this.lastCompressionRatio * 100,
            isKeyFrame: wasKeyFrame,
            encodingMethod: this.lastEncodingMethod,
            viewers: this.viewersCount
        };
    }
}