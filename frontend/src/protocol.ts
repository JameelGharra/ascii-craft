export const FLAG_IS_COMPRESSED = 1 << 0;
export const FLAG_METHOD = 1 << 1;
export const FLAG_IS_DELTA = 1 << 2;

export class BinaryReader {
    private view: DataView;
    private buffer: Uint8Array;
    private offset: number;

    constructor(buffer: ArrayBuffer) {
        this.buffer = new Uint8Array(buffer)
        this.view = new DataView(buffer);
        this.offset = 0;
    }
    readByte(): number {
        return this.view.getUint8(this.offset++);
    }
    readVarint(): number {
        let res = 0;
        let shift = 0;
        while(true) {
            const b = this.buffer[this.offset++];
            res |= (b & 0x7F) << shift;
            if((b & 0x80) == 0) break;
            shift += 7;
        }
        return res;
    }
    readRemaining(): Uint8Array {
        return this.buffer.subarray(this.offset);
    }
    hasMore(): boolean {
        return this.offset < this.buffer.length;
    }
}