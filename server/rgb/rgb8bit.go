package rgb

// quantization r, g having 0-7 as levels and blue having 0-3
func RGBToColor8BitANSII(originalR, originalG, originalB uint8) (r, g, b uint8) {
	// r = uint8((uint16(originalR) * 8) / 256)
	// g = uint8((uint16(originalG) * 8) / 256)
	// b = uint8((uint16(originalB) * 4) / 256)
	r = originalR >> 5
	g = originalG >> 5
	b = originalB >> 6
	return
}

// using this to print ANSII (debug for now)
func Scale8BitToTrueColor(r, g, b uint8) (uint8, uint8, uint8) {
	newR := uint8((uint16(r) * 255) / 7)
	newG := uint8((uint16(g) * 255) / 7)
	newB := uint8((uint16(b) * 255) / 3)
	return newR, newG, newB
}

// pack into single byte
func RGBToColor8Bit(originalR, originalG, originalB uint8) uint8 {
	r, g, b := RGBToColor8BitANSII(originalR, originalG, originalB)
	// 5,6,7 bits red, 2,3,4 bits green, 0,1 bits blue
	return (r << 5) | (g << 2) | b
}

// unpack
func Color8BitToRGB(color8bit uint8) (r, g, b uint8) {
	r = (color8bit >> 5) & 0x07
	g = (color8bit >> 2) & 0x07
	b = color8bit & 0x03
	return
}
