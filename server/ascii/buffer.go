package ascii

import (
	"github.com/JameelGharra/ascii-craft/server/rgb"
	"github.com/JameelGharra/ascii-craft/server/utils"
)

// only colors and 8-bit colors only
func (f *Frame) ToColor8bit(out []byte) {
	utils.Assert(len(out) >= len(f.Pixels), "Output buffer too small for conversion")
	for index, pixel := range f.Pixels {
		out[index] = rgb.RGBToColor8Bit(pixel.R, pixel.G, pixel.B)
	}
}

func Xor(a, b, out []byte) {
	utils.Assert(len(a) == len(b) && len(b) == len(out), "Buffer sizes must match for diffing")
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
}
