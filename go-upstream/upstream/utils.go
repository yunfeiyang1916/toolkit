package upstream

func incr(u *uint32) uint32 {
	*u += 1
	return *u
}

type AtomicBool *bool
