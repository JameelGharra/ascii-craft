export function decodeRLE(src: Uint8Array, dst: Uint8Array): void {
    let srcIdx = 0;
    let dstIdx = 0;
    const srcLen = src.length;
    const dstLen = dst.length;

    if(srcLen % 2 !== 0) { // has to be pairs [count, value]
        throw new Error("Invalid RLE data: length must be even (missing value for count");
    }

    while(srcIdx < srcLen) {
        const count = src[srcIdx++];
        const value = src[srcIdx++];

        if (dstIdx + count > dstLen) {
            throw new Error(`RLE Decode Error: Exceeded destination buffer bounds. Reached ${dstIdx + count}, max is ${dstLen}.`);
        }
        dst.fill(value, dstIdx, dstIdx + count)
        dstIdx += count;
    }
}