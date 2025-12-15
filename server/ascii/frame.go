package ascii

type AsciiPixel struct {
	CharCode uint8
	R        uint8
	G        uint8
	B        uint8
}

type Frame struct {
	Width  uint32
	Height uint32
	Pixels []AsciiPixel
}
