package ascii

import (
	"github.com/JameelGharra/ascii-craft/server/rgb"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

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

func (f *Frame) Planar(out *AsciiFrame) {
	utils.Assert(len(out.Buffer) >= 2*len(f.Pixels), "Output buffer too small for planar conversion")
	count := len(f.Pixels)
	for index, pixel := range f.Pixels {
		out.Buffer[index] = pixel.CharCode
		out.Buffer[index+count] = rgb.RGBToColor8Bit(pixel.R, pixel.G, pixel.B)
	}
}

type AsciiFrame struct {
	Width  uint32
	Height uint32
	Buffer []byte
}

func NewAsciiFrame(width, height uint32) *AsciiFrame {
	return &AsciiFrame{
		Width:  width,
		Height: height,
		Buffer: make([]byte, width*height*2),
	}
}

func (f *AsciiFrame) Push(data []byte) *AsciiFrame {
	copy(f.Buffer, data)
	return f
}

// will return itself after xor params
func (f *AsciiFrame) Xor(a, b *AsciiFrame) *AsciiFrame {

	utils.Assert(a.Width == b.Width && a.Height == b.Height, "Frame sizes must match for diffing")
	for i := 0; i < len(a.Buffer); i++ {
		f.Buffer[i] = a.Buffer[i] ^ b.Buffer[i]
	}
	return f
}
