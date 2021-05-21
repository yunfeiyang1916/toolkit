// +build windows

package circuit

//"errors"

//var errSystem = errors.New("defaultSystem: not init")

type System interface {
	Load1() (float64, error)
}

type defaultSystem struct{}

func (defaultSystem) Load1() (float64, error) {
	return 0.0, nil
}

/*
type mockSystem struct {
	open  int32
	value float64
	def   defaultSystem
}

func (d *mockSystem) Open() {
	atomic.StoreInt32(&d.open, 1)
}

func (d *mockSystem) Close() {
	atomic.StoreInt32(&d.open, 0)
}

func (d *mockSystem) Load1() (float64, error) {
	if atomic.LoadInt32(&d.open) == 1 {
		return d.value, nil
	}
	return d.def.Load1()
}
*/
