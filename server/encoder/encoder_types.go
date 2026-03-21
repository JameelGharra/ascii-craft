package encoder

type EncodingType byte

const (
	NONE EncodingType = iota
	XOR_RLE
	HUFFMAN
	XOR_HUFFMAN
)

// just to make it readable in logs
func (e EncodingType) String() string {
	switch e {
	case NONE:
		return "NONE"
	case XOR_RLE:
		return "XOR_RLE"
	case HUFFMAN:
		return "HUFFMAN"
	case XOR_HUFFMAN:
		return "XOR_HUFFMAN"
	default:
		return "UNKNOWN"
	}
}
