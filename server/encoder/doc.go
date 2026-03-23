// Package encoder implements an adaptive, hybrid compression pipeline for video frames.
//
// It evaluates multiple compression strategies (Raw, XOR+RLE, XOR+Canonical Huffman)
// for each delta frame and dynamically selects the algorithm yielding the smallest
// payload size. It manages the construction of I-Frames and P-Frames for the stream.
package encoder
