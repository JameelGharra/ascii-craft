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
    private frameTimestamps: number[] = [];
    private byteCounts: { ts: number; bytes: number }[] = [];
    private rawFrameSizeBytes: number;
    
    private lastCompressionRatio: number = 0;
    // private lastKeyFrame: boolean = false;
    private lastEncodingMethod: string = "RAW";
    private latchedKeyFrame: boolean = false; // if we get an i-frame display it first then vanish
    private viewersCount: number = 0;

    constructor(videoWidth: number, videoHeight: number) {
        // this assumes that 1 pixel = 1 byte btw
        this.rawFrameSizeBytes = videoWidth * videoHeight;
    }

    public recordFrame(packetSizeBytes: number, isKeyFrame: boolean, encodingMethod: string) {
        const now = performance.now();
        this.frameTimestamps.push(now);
        this.byteCounts.push({ ts: now, bytes: packetSizeBytes });
        const secondsAgo = now - MetricsCollector.PRUNE_FRAME_INTERVAL_MS;
        while (this.frameTimestamps.length >0 && this.frameTimestamps[0] < secondsAgo) {
            this.frameTimestamps.shift();
        }
        while (this.byteCounts.length >0 && this.byteCounts[0].ts < secondsAgo) {
            this.byteCounts.shift();
        }
        this.lastCompressionRatio = Math.max(0, 1 - (packetSizeBytes / this.rawFrameSizeBytes));
        // this.lastKeyFrame = isKeyFrame;
        if (isKeyFrame) {
            this.latchedKeyFrame = true;
        }
        this.lastEncodingMethod = encodingMethod;
    }

    public setViewersCount(count: number) {
        this.viewersCount = count;
    }

    public getSnapshot(): MetricsSnapshot {
        const fps = this.frameTimestamps.length;
        const totalBytes = this.byteCounts.reduce((sum, entry) => sum + entry.bytes, 0);
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