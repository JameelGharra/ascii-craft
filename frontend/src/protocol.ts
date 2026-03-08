export * from "./protocol_constants"; // have to gen it with go

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
        if (this.offset >= this.buffer.length) {
            throw new Error("Protocol Error: Buffer overflow reading byte");
        }
        return this.view.getUint8(this.offset++);
    }
    readSlice(length: number): Uint8Array {
        if (this.offset + length > this.buffer.length) {
            throw new Error(`Protocol Error: Buffer overflow reading slice of length ${length}`);
        }
        const slice = this.buffer.subarray(this.offset, this.offset + length);
        this.offset += length;
        return slice;
    }
    readVarint(): number {
        let res = 0;
        let shift = 0;
        while(true) {
            if (this.offset >= this.buffer.length) {
                throw new Error("Protocol Error: Unexpected end of buffer reading varint");
            }
            const b = this.buffer[this.offset++];
            res |= (b & 0x7F) << shift;
            if((b & 0x80) == 0) break;
            shift += 7;
            if (shift >= 32) {
                throw new Error("Protocol Error: Malformed varint exceeds 32 bits");
            }
        }
        return res >>> 0; // forcing js to treat res as u32int
    }
    readRemaining(): Uint8Array {
        const slice = this.buffer.subarray(this.offset);
        this.offset = this.buffer.length;
        return slice;
    }
    hasMore(): boolean {
        return this.offset < this.buffer.length;
    }
}