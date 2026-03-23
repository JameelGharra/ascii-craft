// Package protocol defines the custom binary protocol used over WebSockets to
// stream video frames to frontend clients.
//
// It includes optimized packet builders, LEB128 Varint encoding, and bit-packed
// metadata flags to strictly minimize network overhead compared to JSON or
// standard text protocols.
package protocol
