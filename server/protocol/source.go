package protocol

type PacketSource interface {
	WriteTo(pb *PacketBuilder) error
}
