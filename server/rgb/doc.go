// Package rgb handles color quantization and bit-packing operations.
//
// To drastically lower the required network bandwidth per frame, it reduces
// 24-bit TrueColor RGB down to 8-bit (3-3-2) or 16-bit representations, allowing
// entire pixel colors to be packed into single bytes.
package rgb
