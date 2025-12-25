package huffman

// won't get to that we are using 16-bit colors, just for experimenting
const MaxHuffmanCodeLength = 24

type HuffmanEncodeResultTable struct {
	ValToCode  map[int][]byte
	CodeMaxLen int
	buffer     []byte
	currentLen int
}

func NewHuffmanEncodeResultTable() *HuffmanEncodeResultTable {
	return &HuffmanEncodeResultTable{
		ValToCode:  make(map[int][]byte),
		CodeMaxLen: 0,
		buffer:     make([]byte, MaxHuffmanCodeLength),
		currentLen: 0,
	}
}

func (h *HuffmanEncodeResultTable) Left() {
	h.buffer[h.currentLen] = 0
	h.currentLen++
}

func (h *HuffmanEncodeResultTable) Right() {
	h.buffer[h.currentLen] = 1
	h.currentLen++
}

func (h *HuffmanEncodeResultTable) Back() { // either way no need to delete anything
	h.currentLen--
}

func (h *HuffmanEncodeResultTable) Update(value int) {
	code := make([]byte, h.currentLen)
	copy(code, h.buffer[:h.currentLen])
	h.ValToCode[value] = code
	if h.currentLen > h.CodeMaxLen {
		h.CodeMaxLen = h.currentLen
	}
}
