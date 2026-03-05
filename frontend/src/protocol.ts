export const FLAG_IS_COMPRESSED = 1 << 0;
export const FLAG_METHOD = 1 << 1;
export const FLAG_IS_DELTA = 1 << 2;

// bits 3 and 4 are for the table mode
export const MASK_TABLE_MODE    = 0x18; // 0001 1000
export const SHIFT_TABLE_MODE   = 3;

export const TABLE_MODE_RAW     = 0;
export const TABLE_MODE_RLE     = 1;
export const TABLE_MODE_SPARSE  = 2;

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
    readSlice(length: number): Uint8Array {
        const slice = this.buffer.subarray(this.offset, this.offset + length);
        this.offset += length;
        return slice;
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