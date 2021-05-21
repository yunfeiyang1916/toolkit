package ikio

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

const (
	negotiationHeaderMagicSize        = 4
	negotiationHeaderFlagsSize        = 4
	negotiationHeaderPeerIDLengthSize = 4
	negotiationHeaderFixedSize        = negotiationHeaderMagicSize + negotiationHeaderFlagsSize + negotiationHeaderPeerIDLengthSize

	packetHeaderIDSize      = 8
	packetHeaderCodeSize    = 4
	packetHeaderFlagsSize   = 4
	packetHeaderHeaderSize  = 4
	packetHeaderPayloadSize = 4
	packetHeaderFixedSize   = packetHeaderIDSize + packetHeaderCodeSize + packetHeaderFlagsSize + packetHeaderHeaderSize + packetHeaderPayloadSize

	packetFlagHeaderZlibEncoded = 1

	HintTypeKeepalive = 0

	negotiationFlagAcceptZlib = 1
	negotiationFlagNoHint     = 2
	negotiationFlagMask       = 0xFFFF
)

const (
	PacketTypeRequest  = 0
	PacketTypeResponse = 1
	PacketTypeHint     = 2
)

// RPC Nego msg type
const (
	RPCNegoMessageType = 0xFFFF
)

// RPC body/header limit
const (
	RPCDefaultMaxHeaderLen = 10 * 1024 * 1024
	RPCDefaultMaxBodyLen   = 10 * 1024 * 1024
	RPCDefaultMaxPeerIDLen = 1024
)

// rpc codec errors
var (
	ErrRPCHeaderSizeLimit = errors.New("rpc header size max error")
	ErrRPCBodySizeLimit   = errors.New("rpc body size max error")
	ErrRPCPeerIDSizeLimit = errors.New("rpc peerid size max error")
)

type RPCPacketHeader struct {
	Key   []byte
	Value []byte
}

type RPCPacket struct {
	ID      int64
	Code    int32
	Header  []RPCPacketHeader
	Payload []byte
	Tp      int32
	Flags   int32

	// timestamp
	T int64
}

var packetPool = sync.Pool{New: func() interface{} { return &RPCPacket{} }}

func GetRPCPacket() *RPCPacket {
	p := packetPool.Get().(*RPCPacket)
	p.reset()
	return p
}

func PutRPCPacket(p *RPCPacket) {
	packetPool.Put(p)
}

func (rp *RPCPacket) reset() {
	rp.ID = 0
	rp.Code = 0
	for _, v := range rp.Header {
		v.Key = v.Key[0:0]
		v.Value = v.Value[0:0]
	}
	rp.Payload = rp.Payload[0:0]
	rp.Tp = 0
	rp.Flags = 0
	rp.T = 0
}

func (rp *RPCPacket) Timestamp() int64 {
	return rp.T
}

func (rp *RPCPacket) AddHeader(key []byte, value []byte) {
	if rp.Header == nil {
		rp.Header = []RPCPacketHeader{}
	}
	rp.Header = append(rp.Header, RPCPacketHeader{key, value})
}

func (rp *RPCPacket) GetHeader(key []byte) ([]byte, bool) {
	for _, header := range rp.Header {
		if bytes.Equal(header.Key, key) {
			return header.Value, true
		}
	}
	return nil, false
}

func (rp *RPCPacket) GetHeaderUint64(key []byte) (uint64, bool) {
	v, ok := rp.GetHeader(key)
	if !ok {
		return 0, false
	}
	return binary.BigEndian.Uint64(v), true
}

func (rp *RPCPacket) GetHeaderUint32(key []byte) (uint32, bool) {
	v, ok := rp.GetHeader(key)
	if !ok {
		return 0, false
	}
	return binary.BigEndian.Uint32(v), true
}

func (rp *RPCPacket) ForeachHeader(cb func(key, value []byte) error) error {
	for _, h := range rp.Header {
		if err := cb(h.Key, h.Value); err != nil {
			return err
		}
	}
	return nil
}

func (rp *RPCPacket) serializePacketHeader() []byte {
	lenBuffer := make([]byte, 4)
	var buffer bytes.Buffer
	if rp.Header == nil || len(rp.Header) == 0 {
		return nil
	}
	for _, header := range rp.Header {
		key := header.Key
		binary.BigEndian.PutUint32(lenBuffer, uint32(len(key)))
		buffer.Write(lenBuffer)
		buffer.Write(key)
		buffer.WriteByte(0)
		value := header.Value
		binary.BigEndian.PutUint32(lenBuffer, uint32(len(value)))
		buffer.Write(lenBuffer)
		buffer.Write(value)
		buffer.WriteByte(0)
	}
	return buffer.Bytes()
}

func (rp *RPCPacket) Type() int32 {
	return rp.Tp
}
func (rp *RPCPacket) Serialize() ([]byte, error) {
	header := rp.serializePacketHeader()
	headerLength := len(header)
	payloadLength := len(rp.Payload)
	var length = packetHeaderFixedSize + headerLength + payloadLength
	buffer := make([]byte, length)
	binary.BigEndian.PutUint64(buffer[0:8], uint64(rp.ID))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(rp.Code))
	binary.BigEndian.PutUint32(buffer[12:16], uint32(rp.Flags<<8|rp.Tp))
	binary.BigEndian.PutUint32(buffer[16:20], uint32(headerLength))
	binary.BigEndian.PutUint32(buffer[20:24], uint32(payloadLength))
	if headerLength > 0 {
		copy(buffer[24:length], header)
	}
	if payloadLength > 0 {
		copy(buffer[24+headerLength:length], rp.Payload)
	}
	return buffer, nil
}

type RPCNegoPacket struct {
	Magic uint32
	ID    []byte
	Flag  uint32
}

func (rnp *RPCNegoPacket) Timestamp() int64 {
	return 0
}

func (rnp *RPCNegoPacket) Serialize() ([]byte, error) {
	n := negotiationHeaderFixedSize + len(rnp.ID) + 1
	buffer := make([]byte, n)
	binary.BigEndian.PutUint32(buffer, rnp.Magic)
	binary.BigEndian.PutUint32(buffer[4:8], uint32(0))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(len(rnp.ID)))
	copy(buffer[12:n], rnp.ID)
	return buffer, nil
}

func (rnp *RPCNegoPacket) Type() int32 {
	return RPCNegoMessageType
}

type RPCCodec struct {
	negoDone     bool
	MaxHeaderLen int32
	MaxBodyLen   int32
	MaxPeerIDLen int32
}

func (rc *RPCCodec) Encode(p Packet, writer io.Writer) (int, error) {
	data, err := p.Serialize()
	if err != nil {
		return 0, err
	}
	return writer.Write(data)
}

func (rc *RPCCodec) Decode(reader *bufio.Reader) (Packet, error) {
	if !rc.negoDone {
		negotiationHeader := make([]byte, negotiationHeaderFixedSize)
		_, err := io.ReadFull(reader, negotiationHeader)
		if err != nil {
			return nil, err
		}
		pkt := new(RPCNegoPacket)
		pkt.Magic = binary.BigEndian.Uint32(negotiationHeader[:4])
		pkt.Flag = binary.BigEndian.Uint32(negotiationHeader[4:8])
		n := binary.BigEndian.Uint32(negotiationHeader[8:])
		if (rc.MaxPeerIDLen == 0 && n > uint32(RPCDefaultMaxPeerIDLen)) || (rc.MaxPeerIDLen != 0 && n > uint32(rc.MaxPeerIDLen)) {
			return nil, ErrRPCPeerIDSizeLimit
		}
		peerID := make([]byte, n+1)
		_, err = io.ReadFull(reader, peerID)
		if err != nil {
			return nil, err
		}
		pkt.ID = peerID[:n]
		rc.negoDone = true
		return pkt, nil

	}
	buffer := make([]byte, packetHeaderFixedSize)
	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		return nil, err
	}
	pkt := GetRPCPacket()
	pkt.ID = int64(binary.BigEndian.Uint64(buffer[:8]))
	pkt.Code = int32(binary.BigEndian.Uint32(buffer[8:12]))
	pkt.Flags = int32(binary.BigEndian.Uint32(buffer[12:]))
	pkt.Tp = pkt.Flags & 0xFF
	pkt.Flags >>= 8

	headerSize := int32(binary.BigEndian.Uint32(buffer[16:20]))
	if headerSize < 0 ||
		(rc.MaxHeaderLen == 0 && headerSize > RPCDefaultMaxHeaderLen) || (rc.MaxHeaderLen != 0 && rc.MaxHeaderLen < headerSize) {
		return nil, ErrRPCHeaderSizeLimit
	}
	payloadSize := int32(binary.BigEndian.Uint32(buffer[20:24]))
	if payloadSize < 0 ||
		(rc.MaxBodyLen == 0 && payloadSize > RPCDefaultMaxBodyLen) || (rc.MaxBodyLen != 0 && rc.MaxBodyLen < payloadSize) {
		return nil, ErrRPCBodySizeLimit
	}

	headerBuffer := make([]byte, headerSize)
	_, err = io.ReadFull(reader, headerBuffer)
	if err != nil {
		return pkt, err
	}
	pkt.Header, err = parseHeader(headerBuffer)
	if err != nil {
		return pkt, err
	}
	payloadBuffer := make([]byte, payloadSize)
	_, err = io.ReadFull(reader, payloadBuffer)
	if err != nil {
		return pkt, err
	}
	pkt.Payload = payloadBuffer[:payloadSize]
	return pkt, err
}

func parseHeader(header []byte) ([]RPCPacketHeader, error) {
	headers := make([]RPCPacketHeader, 0)
	for {
		l := int32(len(header))
		if l < 4 {
			break
		}
		keyLen := int32(binary.BigEndian.Uint32(header[:4]))
		valueLenStart := 4 + keyLen + 1
		if l < valueLenStart+4 {
			break
		}
		valueLen := int32(binary.BigEndian.Uint32(header[valueLenStart:]))
		if l < 4+valueLenStart+valueLen+1 {
			break
		}
		h := RPCPacketHeader{
			Key:   header[4 : 4+keyLen],
			Value: header[4+valueLenStart : 4+valueLenStart+valueLen],
		}
		headers = append(headers, h)
		header = header[4+valueLenStart+valueLen+1:]
	}
	return headers, nil
}
