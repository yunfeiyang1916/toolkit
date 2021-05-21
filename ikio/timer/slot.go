package timer

type slotValueType struct{}

type slot struct {
	p map[*Context]slotValueType
}

func newSlots(size int) []*slot {
	slots := make([]*slot, size)
	for i := 0; i < size; i++ {
		slots[i] = &slot{
			p: make(map[*Context]slotValueType),
		}
	}
	return slots
}

func (s *slot) delTimer(value *Context) {
	delete(s.p, value)
}

func (s *slot) add(value *Context) {
	s.p[value] = slotValueType{}
}

func (s *slot) foreach(cb func(*Context)) {
	for v := range s.p {
		cb(v)
	}
}
