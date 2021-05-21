package timer

import (
	"sync"
	"time"

	metrics "github.com/yunfeiyang1916/toolkit/metrics"
	"golang.org/x/net/context"
)

type TimerId uint64

// TimingWheel manages all the timed task.
type TimingWheel struct {
	opts        *Options
	timeOutChan chan *Context
	ticker      *time.Ticker
	wg          *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc

	currSlot int
	slots    []*slot

	mu sync.Mutex
}

// NewTimingWheel returns a *TimingWheel ready for use.
func NewTimingWheel(ctx context.Context, setters ...Option) *TimingWheel {
	opts := &Options{
		SlotSize:  defaultSlotSize,
		Precision: defaultPrecision,
		BufSize:   defualtBufSize,
	}
	for _, setter := range setters {
		setter(opts)
	}
	timingWheel := &TimingWheel{
		opts:        opts,
		timeOutChan: make(chan *Context, opts.BufSize),
		ticker:      time.NewTicker(opts.Precision),
		wg:          &sync.WaitGroup{},
	}
	timingWheel.ctx, timingWheel.cancel = context.WithCancel(ctx)
	timingWheel.slots = newSlots(opts.SlotSize)

	timingWheel.wg.Add(1)
	go func() {
		timingWheel.start()
		timingWheel.wg.Done()
	}()
	return timingWheel
}

// TimeOutChannel returns the timeout channel.
func (tw *TimingWheel) TimeOutChannel() chan *Context {
	return tw.timeOutChan
}

func WithValue(value interface{}) func(c *Context) {
	return func(c *Context) {
		c.values = value
	}
}

func (tw *TimingWheel) CancelTimer(ctx *Context) {
	tw.mu.Lock()
	tw.slots[ctx.slot].delTimer(ctx)
	tw.mu.Unlock()
}

// AddTimer adds new timed task.
func (tw *TimingWheel) AddTimer(after, interval time.Duration, setters ...func(c *Context)) (*Context, error) {
	ctx := &Context{
		interval: interval,
	}
	for _, setter := range setters {
		setter(ctx)
	}
	index, circle := tw.getSlotAndCircle(after)

	tw.mu.Lock()
	tw.safeAddTimer(ctx, index, circle)
	tw.mu.Unlock()
	return ctx, nil
}

func (tw *TimingWheel) getSlotAndCircle(after time.Duration) (pos int, circle int) {
	if after < tw.opts.Precision {
		after = tw.opts.Precision
	}
	circle = int(after/tw.opts.Precision) / tw.opts.SlotSize
	pos = (tw.currSlot + int(after/tw.opts.Precision)) % tw.opts.SlotSize
	return
}

func (tw *TimingWheel) safeAddTimer(ctx *Context, index, circle int) {
	ctx.slot = index
	ctx.circle = circle
	tw.slots[index].add(ctx)
}

func (tw *TimingWheel) start() {
	defer tw.ticker.Stop()
	for {
		select {
		case <-tw.ctx.Done():
			return
		case <-tw.ticker.C:
			tw.mu.Lock()
			slot := tw.slots[tw.currSlot]
			fired := make([]*Context, 0, 1024)
			slot.foreach(func(c *Context) {
				if c.circle > 0 {
					c.circle--
					return
				}
				fired = append(fired, c)
			})
			// 删除oneshot触发的定期是
			for _, value := range fired {
				slot.delTimer(value)
				if value.interval <= 0 {
					continue
				}
				index, circle := tw.getSlotAndCircle(value.interval)
				tw.safeAddTimer(value, index, circle)
			}
			tw.currSlot = (tw.currSlot + 1) % tw.opts.SlotSize
			total := 0
			for i := 0; i < tw.opts.SlotSize; i++ {
				total += len(tw.slots[i].p)
			}
			tw.mu.Unlock()

			beginTime := time.Now()
			for _, value := range fired {
				tw.timeOutChan <- value
			}
			metrics.Timer("ikio.timer.ticker-block", beginTime, tw.opts.metricsTags...)
			metrics.Meter("ikio.timer.ticker-fired", len(fired), tw.opts.metricsTags...)
		}
	}
}

// Stop stops the TimingWheel.
func (tw *TimingWheel) Stop() {
	tw.cancel()
	tw.wg.Wait()
}
