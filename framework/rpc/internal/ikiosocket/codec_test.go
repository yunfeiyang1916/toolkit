package ikiosocket

import (
	"encoding/binary"
	//"testing"
)

var (
	UIDKey    = []byte("uid")
	FlowIDKey = []byte("flowid")
	CMDKey    = []byte("cmd")
)

type UAMessage struct {
	UID, FlowID uint64
	CMD         int32
	Body        []byte
}

func (m UAMessage) Serialize() ([]byte, error) {
	return m.Body, nil
}

func (m UAMessage) Packet(pkt *RPCPacket) {
	buffer := make([]byte, 20)
	binary.BigEndian.PutUint64(buffer[0:8], m.UID)
	binary.BigEndian.PutUint64(buffer[8:16], m.FlowID)
	binary.BigEndian.PutUint32(buffer[16:20], uint32(m.CMD))
	pkt.AddHeader(UIDKey, buffer[0:8])
	pkt.AddHeader(FlowIDKey, buffer[8:16])
	pkt.AddHeader(CMDKey, buffer[16:20])
}
