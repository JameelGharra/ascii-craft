export function decodeRLE(src: Uint8Array, dst: Uint8Array): void {
    let srcIdx = 0;
    let dstIdx = 0;
    const len = src.length;

    while(srcIdx < len) {
        const count = src[srcIdx++];
        const value = src[srcIdx++];

        dst.fill(value, dstIdx, dstIdx + count)
        dstIdx += count;
    }
}