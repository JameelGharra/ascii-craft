// src/decoder/huffman.ts
import { TABLE_MODE_RAW, TABLE_MODE_RLE, TABLE_MODE_SPARSE } from "../protocol";

// Simple tree node for decoding
class Node {
    left: Node | null = null;
    right: Node | null = null;
    symbol: number = -1; // -1 means internal node, 0-255 is a leaf
    isLeaf: boolean = false;
}

export class HuffmanDecoder {
    // We reuse this array to store code lengths (index = symbol, value = length)
    private codeLengths = new Uint8Array(256);
    private root: Node = new Node();

    // 1. Reconstruct Code Lengths from Meta Data
    // Returns the new offset in the buffer (how many bytes consumed)
    public decodeTable(mode: number, buffer: Uint8Array): void {
        this.codeLengths.fill(0);

        if (mode === TABLE_MODE_RAW) {
            // Raw: 256 bytes, index maps to symbol
            for (let i = 0; i < 256; i++) {
                this.codeLengths[i] = buffer[i];
            }
        } else if (mode === TABLE_MODE_RLE) {
            // RLE: [Count, LengthValue], [Count, LengthValue]...
            let ptr = 0;
            let symbolIdx = 0;
            while (ptr < buffer.length && symbolIdx < 256) {
                const count = buffer[ptr++];
                const lengthVal = buffer[ptr++];
                
                // Fill 'count' symbols with 'lengthVal'
                const end = symbolIdx + count;
                while (symbolIdx < end && symbolIdx < 256) {
                    this.codeLengths[symbolIdx++] = lengthVal;
                }
            }
        } else if (mode === TABLE_MODE_SPARSE) {
            // Sparse: [Symbol, Length], [Symbol, Length]...
            let ptr = 0;
            while (ptr < buffer.length) {
                const symbol = buffer[ptr++];
                const lengthVal = buffer[ptr++];
                this.codeLengths[symbol] = lengthVal;
            }
        }
        
        // After getting lengths, rebuild the tree immediately
        this.buildCanonicalTree();
    }

    // 2. Build the Canonical Huffman Tree
    // This logic matches Go's `generateCanonicalCodes`
    private buildCanonicalTree() {
        this.root = new Node(); // Reset tree

        // Step A: Sort symbols by length
        // max depth is usually small (<32)
        const maxLen = 32;
        const bl_count = new Int32Array(maxLen + 1);
        const next_code = new Int32Array(maxLen + 1);

        // Count how many codes of each length
        for (let i = 0; i < 256; i++) {
            const len = this.codeLengths[i];
            if (len > 0) bl_count[len]++;
        }

        // Calculate starting code for each length
        let code = 0;
        bl_count[0] = 0;
        for (let bits = 1; bits <= maxLen; bits++) {
            code = (code + bl_count[bits - 1]) << 1;
            next_code[bits] = code;
        }

        // Step B: Assign codes and build the tree
        // We iterate 0..255 so that for same length, symbols are naturally sorted
        for (let i = 0; i < 256; i++) {
            const len = this.codeLengths[i];
            if (len === 0) continue;

            const myCode = next_code[len];
            next_code[len]++; // Increment for next symbol of this length

            this.insertToTree(myCode, len, i);
        }
    }

    private insertToTree(code: number, len: number, symbol: number) {
        let node = this.root;
        // In Canonical Huffman, we read bits from MSB to LSB relative to length
        // e.g. code 110 (6), len 3. 
        // Bit 2: (6 >> 2) & 1 = 1
        // Bit 1: (6 >> 1) & 1 = 1
        // Bit 0: (6 >> 0) & 1 = 0
        for (let i = len - 1; i >= 0; i--) {
            const bit = (code >> i) & 1;
            if (bit === 0) {
                if (!node.left) node.left = new Node();
                node = node.left;
            } else {
                if (!node.right) node.right = new Node();
                node = node.right;
            }
        }
        node.isLeaf = true;
        node.symbol = symbol;
    }

    // 3. Decode the Bitstream
    public decodeStream(payload: Uint8Array, dst: Uint8Array): void {
        let dstIdx = 0;
        let byteIdx = 0;
        let bitPos = 7; // Start at MSB (7) down to 0
        let node = this.root;
        
        const payloadLen = payload.length;
        const dstLen = dst.length;

        // Loop until we filled the destination buffer
        while (dstIdx < dstLen && byteIdx < payloadLen) {
            
            // Read one bit
            const bit = (payload[byteIdx] >> bitPos) & 1;
            bitPos--;
            if (bitPos < 0) {
                bitPos = 7;
                byteIdx++;
            }

            // Traverse Tree
            if (bit === 0) {
                if(!node.left) throw new Error("Huffman Decode Error: reached deadend on left branch.");
                node = node.left!; // The tree is complete for the data, ! is safe
            } else {
                if(!node.right) throw new Error("Huffman Decode Error: reached deadend on right branch.");
                node = node.right!;
            }

            // Found a symbol
            if (node.isLeaf) {
                dst[dstIdx++] = node.symbol;
                node = this.root; // Reset to root for next symbol
            }
        }
    }
}