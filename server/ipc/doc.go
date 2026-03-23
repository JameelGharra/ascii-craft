// Package ipc provides a high-performance, zero-copy inter-process communication
// layer using shared memory (mmap/MapViewOfFile).
//
// It bridges the C-based OpenGL game engine and the Go orchestration server,
// utilizing atomic lock-free ring buffers for real-time command dispatch and
// video frame retrieval without the overhead of standard I/O pipes.
package ipc
