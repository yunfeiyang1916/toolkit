package ikio

import (
	"bufio"
	"io"
)

type codecNewFunc func() Codec

// 网络读包
type Codec interface {
	Decode(*bufio.Reader) (Packet, error)
	Encode(Packet, io.Writer) (int, error)
}

// 网络协议包
type Packet interface {
	Type() int32
	Serialize() ([]byte, error)
}
