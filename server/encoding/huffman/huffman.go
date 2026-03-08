package huffman

import (
	"errors"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

const (
	// adapting canonical atm
	TableModeRaw                byte = tableModeRaw
	TableModeRLE                byte = tableModeRLE
	TableModeSparse             byte = tableModeSparse
	TableRawSize                     = canonicalTableRawSize
	MinimumAllowingHuffmanChars      = 2 // not really used since hybrid approach
)

var (
	ErrHuffmanValueNotFound       = errors.New("huffman value not found in encode table")
	ErrHuffmanExceedsOutputBuffer = errors.New("huffman encoding exceeds output buffer size")
	ErrHuffmanDecodeTraverse      = errors.New("huffman decode traverser error")
	ErrNoNeedHuffmanEncoding      = errors.New("no need for huffman encoding, data too small")
	ErrHuffmanFailedToWrite       = errors.New("huffman failed to write to output")
	ErrHuffmanPackingDoesntFit    = errors.New("huffman packing doesn't fit in output buffer")
	ErrHuffmanPackingFailure      = errors.New("huffman packing failure")
)

type Huffman struct {
	encodeTable            map[int][]byte
	treeDecodeTraverser    *HuffmanTreeDecodeTraverser
	ValToCodeLength        map[int]int
	cachedPacketTableBytes []byte
	cachedMetaMode         byte
}

func NewHuffman(freqTable *ascii.FreqTable) (*Huffman, error) {
	// if freqTable.TotalDifferentChars < MinimumAllowingHuffmanChars {
	// 	return nil, ErrNoNeedHuffmanEncoding
	// }
	huffmanTreeRoot, err := buildEncodeTree(freqTable)
	if err != nil {
		return nil, err
	}
	valToCodeLength := make(map[int]int)
	huffmanTreeRoot.calculateDepths(0, valToCodeLength)
	valToCode := generateCanonicalCodes(valToCodeLength)
	decodeTreeRoot := buildCanonicalDecodeTree(valToCode)
	return (&Huffman{
		encodeTable:            valToCode,
		treeDecodeTraverser:    NewHuffmanTreeDecodeTraverser(decodeTreeRoot),
		ValToCodeLength:        valToCodeLength,
		cachedPacketTableBytes: nil,
		cachedMetaMode:         0,
	}), nil
}

// will return the number of bits written
func (h *Huffman) Encode(dataToEncode utils.IByteIterator, out []byte) (int, error) {
	byteIndexShift, currentEncoding := 7, byte(0)
	outIndex, dirty, bitLength := 0, false, 0
	for dataToEncode.HasNext() {
		value := dataToEncode.Next()
		huffmanCode, exists := h.encodeTable[value]
		if !exists {
			return 0, ErrHuffmanValueNotFound
		}
		for _, bit := range huffmanCode { // msb to lsb here
			dirty = true
			currentEncoding = currentEncoding | bit<<byteIndexShift
			byteIndexShift--
			bitLength++
			if byteIndexShift < 0 {
				if outIndex >= len(out) {
					return 0, ErrHuffmanExceedsOutputBuffer
				}
				out[outIndex] = currentEncoding
				outIndex++
				currentEncoding = 0
				dirty = false
				byteIndexShift = 7
			}
		}
	}
	if dirty { // padding
		if outIndex >= len(out) {
			return 0, ErrHuffmanExceedsOutputBuffer
		}
		out[outIndex] = currentEncoding
	}
	return bitLength, nil
}

// would need this in client side, but decided to implement it anyway being hyped
func (h *Huffman) Decode(dataToDecode []byte, encodingBitLen int, writer utils.ByteWriter) error {
	currentBitIndex := 7
	var bitDirection byte
	stillOnIt := true
	for _, encodedByte := range dataToDecode {
		for currentBitIndex >= 0 && encodingBitLen > 0 {
			bitDirection = (encodedByte >> currentBitIndex) & 1
			isLeaf, value, err := h.treeDecodeTraverser.TraverseStep(bitDirection)
			if err != nil {
				return errors.Join(ErrHuffmanDecodeTraverse, err)
			}
			stillOnIt = true
			if isLeaf {
				err = writer.Write(value)
				if err != nil {
					return errors.Join(ErrHuffmanFailedToWrite, err)
				}
				stillOnIt = false
			}
			currentBitIndex--
			encodingBitLen--
		}
		currentBitIndex = 7
	}
	if stillOnIt {
		return ErrHuffmanDecodeTraverse
	}
	return nil
}

// gets the table size to be stored inside the packet later on
func (h *Huffman) GetTableSize() (int, error) {
	if h.cachedPacketTableBytes != nil {
		size := len(h.cachedPacketTableBytes)
		if h.cachedMetaMode != tableModeRaw {
			size++ // prefix meta byte
		}
		return size, nil
	}
	metaMode, packedData, err := packCanonicalTable(h.ValToCodeLength)
	if err != nil {
		return 0, err
	}
	size := len(packedData)
	h.cachedPacketTableBytes = packedData
	h.cachedMetaMode = metaMode
	if metaMode != tableModeRaw {
		size++
	}
	return size, nil
}

func (h *Huffman) WriteTableBytes(dst []byte) (byte, int, error) {
	if h.cachedPacketTableBytes == nil {
		_, err := h.GetTableSize()
		if err != nil {
			return 0, 0, errors.Join(ErrHuffmanPackingFailure, err)
		}
	}
	bytesToWrite := len(h.cachedPacketTableBytes)
	offset := 0
	if h.cachedMetaMode != tableModeRaw {
		if len(dst) < 1 {
			return 0, 0, ErrHuffmanPackingDoesntFit
		}
		dst[0] = byte(len(h.cachedPacketTableBytes))
		bytesToWrite++
		offset++
	}
	if bytesToWrite > len(dst) {
		return 0, 0, ErrHuffmanPackingDoesntFit
	}
	copy(dst[offset:], h.cachedPacketTableBytes)
	return h.cachedMetaMode, bytesToWrite, nil
}
