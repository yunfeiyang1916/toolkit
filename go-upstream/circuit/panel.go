package circuit

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/olekukonko/tablewriter"
	log "github.com/yunfeiyang1916/toolkit/logging"
	metrics "github.com/yunfeiyang1916/toolkit/metrics"
)

var globalPanel = NewPanel()

func AddBreaker(name string, cb *Breaker) {
	globalPanel.Add(name, cb)
}

func GetBreaker(name string) *Breaker {
	return globalPanel.Get(name)
}

var logger atomic.Value

func SetLogger(l *log.Logger) {
	logger.Store(l)
}

var defaultStatsPrefixf = "circuit.%s"

// PanelEvent wraps a BreakerEvent and provides the string name of the breaker
type PanelEvent struct {
	Name  string
	Event BreakerEvent
}

// Panel tracks a group of circuit breakers by name.
type Panel struct {
	StatsPrefixf string

	circuitBreakers *sync.Map

	lastTripTimes  map[string]time.Time
	tripTimesLock  sync.RWMutex
	panelLock      sync.RWMutex
	eventReceivers []chan PanelEvent
}

// NewPanel creates a new Panel
func NewPanel() *Panel {
	p := &Panel{
		circuitBreakers: new(sync.Map),
		StatsPrefixf:    defaultStatsPrefixf,
		lastTripTimes:   make(map[string]time.Time),
	}
	c := time.Tick(120 * time.Second)
	go func() {
		for {
			<-c
			b := &bytes.Buffer{}
			table := tablewriter.NewWriter(b)
			table.SetHeader([]string{"Date", "Resource", "Triped", "Cause", "Concurrent", "Load1", "AVG(MS)", "Percent", "Samples", "Consecutive"})
			p.circuitBreakers.Range(func(name interface{}, value interface{}) bool {
				cb := value.(*Breaker)
				stat := cb.GetStats()
				c := []string{}
				c = append(c, time.Now().Format("2006-01-02 15:04:05"))
				c = append(c, name.(string))
				c = append(c, fmt.Sprintf("%v", cb.Tripped()))
				c = append(c, fmt.Sprintf("%v", cb.trippedError.Load()))
				c = append(c, fmt.Sprintf("%v", stat.Concurrent))
				c = append(c, fmt.Sprintf("%v", stat.SystemLoad))
				c = append(c, fmt.Sprintf("%v", int(stat.AverageRT/time.Millisecond)))
				c = append(c, fmt.Sprintf("%v", stat.ErrorPercent))
				c = append(c, fmt.Sprintf("%v", stat.ErrorSamples))
				c = append(c, fmt.Sprintf("%v", stat.ErrorConsecutive))
				table.Append(c)
				return true
			})
			table.Render()
			log.GenLogf("(Circuit Status Information)\n%s", b.Bytes())
		}
	}()

	return p
}

// Add sets the name as a reference to the given circuit breaker.
func (p *Panel) Add(name string, cb *Breaker) {
	if _, ok := p.circuitBreakers.LoadOrStore(name, cb); ok {
		return
	}
	events := cb.Subscribe()
	go func() {
		for event := range events {
			for _, receiver := range p.eventReceivers {
				receiver <- PanelEvent{name, event}
			}
			switch event {
			case BreakerTripped:
				p.breakerTripped(name)
			case BreakerReset:
				p.breakerReset(name)
			case BreakerFail:
				p.breakerFail(name)
			case BreakerReady:
				p.breakerReady(name)
			}
		}
	}()
}

// Get retrieves a circuit breaker by name.  If no circuit breaker exists, it
// returns the NoOp one and sets ok to false.
func (p *Panel) Get(name string) *Breaker {
	if _, ok := p.circuitBreakers.Load(name); !ok {
		p.panelLock.Lock()
		defer p.panelLock.Unlock()

		if val, ok := p.circuitBreakers.Load(name); ok {
			return val.(*Breaker)
		}
		b := NewBreakerWithOptions(&Options{Name: name})
		p.Add(name, b)
		return b
	}
	b, _ := p.circuitBreakers.Load(name)
	return b.(*Breaker)
}

// Subscribe returns a channel of PanelEvents. Whenever a breaker changes state,
// the PanelEvent will be sent over the channel. See BreakerEvent for the types of events.
func (p *Panel) Subscribe() <-chan PanelEvent {
	eventReader := make(chan PanelEvent)
	output := make(chan PanelEvent, 100)

	go func() {
		for v := range eventReader {
			select {
			case output <- v:
			default:
				<-output
				output <- v
			}
		}
	}()
	p.eventReceivers = append(p.eventReceivers, eventReader)
	return output
}

func (p *Panel) breakerTripped(name string) {
	metrics.CounterInc(
		fmt.Sprintf(p.StatsPrefixf, name)+".tripped",
		"event", "tripped", "resource", name,
	)
	p.tripTimesLock.Lock()
	p.lastTripTimes[name] = time.Now()
	p.tripTimesLock.Unlock()
	log.GenLogf("circuit: resource %s get tripped", name)
}

func (p *Panel) breakerReset(name string) {
	bucket := fmt.Sprintf(p.StatsPrefixf, name)

	metrics.CounterInc(
		bucket+".reset",
		"event", "reset", "resource", name,
	)

	p.tripTimesLock.RLock()
	lastTrip := p.lastTripTimes[name]
	p.tripTimesLock.RUnlock()

	if !lastTrip.IsZero() {
		p.tripTimesLock.Lock()
		p.lastTripTimes[name] = time.Time{}
		p.tripTimesLock.Unlock()
	}
	log.GenLogf("circuit: resource %s get reset", name)
}

func (p *Panel) breakerFail(name string) {
	metrics.CounterInc(
		fmt.Sprintf(p.StatsPrefixf, name)+".fail",
		"event", "fail", "resource", name,
	)
}

func (p *Panel) breakerReady(name string) {
	metrics.CounterInc(
		fmt.Sprintf(p.StatsPrefixf, name)+".ready",
		"event", "ready", "resource", name,
	)
	log.GenLogf("circuit: resource %s get ready", name)
}
