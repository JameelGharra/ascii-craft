package encoding

import (
	"log"

	"github.com/JameelGharra/ascii-craft/server/ascii"
	"github.com/JameelGharra/ascii-craft/server/rgb"
)

// seperating ascii from 8-bit colors having ascii in first portion
func SeperateCharsColor(input []ascii.AsciiPixel, out []byte) {
	count := len(input)
	if len(out) < count*2 {
		log.Fatal("Incorrect output buffer size for planar separation")
		return
	}
	for index, pixel := range input {
		out[index] = pixel.CharCode
		out[index+count] = rgb.RGBToColor8Bit(pixel.R, pixel.G, pixel.B)
	}
}

//
