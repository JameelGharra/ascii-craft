package rgb

// quantization r, g having 0-63 as levels and blue having 0-31
func RGBToColor16BitANSII(originalR, originalG, originalB uint8) (r, g, b uint8) {
	r = originalR >> 2
	g = originalG >> 2
	b = originalB >> 3
	return
}

// using this to print ANSII (debug for now)
func Scale16BitToTrueColor(r, g, b uint8) (uint8, uint8, uint8) {
	newR := uint8((uint16(r) * 255) / 63)
	newG := uint8((uint16(g) * 255) / 63)
	newB := uint8((uint16(b) * 255) / 31)
	return newR, newG, newB
}

// pack into two bytes
func RGBToColor16Bit(originalR, originalG, originalB uint8) uint16 {
	r, g, b := RGBToColor16BitANSII(originalR, originalG, originalB)
	return (uint16(r) << 11) | (uint16(g) << 5) | uint16(b)
}

// unpack
func Color16BitToRGB(color16bit uint16) (r, g, b uint8) {
	r = uint8((color16bit >> 11) & 0x3F)
	g = uint8((color16bit >> 5) & 0x3F)
	b = uint8(color16bit & 0x1F)
	return
}
