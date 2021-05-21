package ikiosocket

import (
	"sync"
)

var codecPool = &sync.Pool{
	New: func() interface{} {
		return new(RPCPacket)
	},
}

func Get() *RPCPacket {
	return codecPool.Get().(*RPCPacket)
}

func Put(p *RPCPacket) {
	p.ID = 0
	p.Code = 0
	p.Flags = 0
	p.Tp = 0
	p.Header = nil
	p.Payload = nil
	codecPool.Put(p)
}
