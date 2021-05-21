package ikio

import (
	"bufio"
	"io"
)

type LinePacket struct {
	Payload []byte
}

func (lp *LinePacket) Type() int32 {
	return 0
}

func (lp *LinePacket) Serialize() ([]byte, error) {
	return lp.Payload, nil
}

func (lp *LinePacket) Timestamp() int64 {
	return 0
}

type LineCodec struct{}

func (lc *LineCodec) Encode(p Packet, writer io.Writer) (int, error) {
	data, err := p.Serialize()
	if err != nil {
		return 0, err
	}
	n, err := writer.Write(data)
	if err != nil {
		return n, err
	}
	m, err := writer.Write([]byte("\r\n"))
	if err != nil {
		return m + n, err
	}
	return m + n, nil
}

func (lc *LineCodec) Decode(reader *bufio.Reader) (Packet, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	l := len(line)
	if l >= 2 {
		if line[l-2] == '\r' {
			line = line[:l-2]
		} else {
			line = line[:l-1]
		}
	} else {
		line = line[:l-1]
	}
	return &LinePacket{Payload: line}, err
}
