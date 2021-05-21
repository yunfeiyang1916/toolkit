// Package circuit implements the Circuit Breaker pattern. It will wrap
// a function call (typically one which uses remote services) and monitors for
// failures and/or time outs. When a threshold of failures or time outs has been
// reached, future calls to the function will not run. During this state, the
// breaker will periodically allow the function to run and, if it is successful,
// will start running the function again.
//
// Circuit includes three types of circuit breakers:
//
// A Threshold Breaker will trip when the failure count reaches a given threshold.
// It does not matter how long it takes to reach the threshold and the failures do
// not need to be consecutive.
//
// A Consecutive Breaker will trip when the consecutive failure count reaches a given
// threshold. It does not matter how long it takes to reach the threshold, but the
// failures do need to be consecutive.
//
//
// When wrapping blocks of code with a Breaker's Call() function, a time out can be
// specified. If the time out is reached, the breaker's Fail() function will be called.
//
//
// Other types of circuit breakers can be easily built by creating a Breaker and
// adding a custom TripFunc. A TripFunc is called when a Breaker Fail()s and receives
// the breaker as an argument. It then returns true or false to indicate whether the
// breaker should trip.
//
// The package also provides a wrapper around an http.Client that wraps all of
// the http.Client functions with a Breaker.
//
package circuit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenk/backoff"
	"github.com/facebookgo/clock"
)

// BreakerEvent indicates the type of event received over an event channel
type BreakerEvent int

const (
	// BreakerTripped is sent when a breaker trips
	BreakerTripped BreakerEvent = iota

	// BreakerReset is sent when a breaker resets
	BreakerReset BreakerEvent = iota

	// BreakerFail is sent when Fail() is called
	BreakerFail BreakerEvent = iota

	// BreakerReady is sent when the breaker enters the half open state and is ready to retry
	BreakerReady BreakerEvent = iota
)

// ListenerEvent includes a reference to the circuit breaker and the event.
type ListenerEvent struct {
	CB    *Breaker
	Event BreakerEvent
}

type state int

const (
	open     state = iota
	halfopen state = iota
	closed   state = iota
)

var (
	defaultInitialBackOffInterval = 500 * time.Millisecond
	defaultBackoffMaxElapsedTime  = 0 * time.Second
)

// Error codes returned by Call
var (
	ErrBreakerTimeout = NewBreakerError("breaker: timeout")

	ErrMaxConcurrent = NewBreakerError("breaker: exceeds max concurrent request number")
	ErrRateLimit     = NewBreakerError("breaker: exceeds request rate limit")
	ErrSystemLoad    = NewBreakerError("breaker: exceeds system load")
	ErrAverageRT     = NewBreakerError("breaker: exceeds average rt")
	ErrConsecutive   = NewBreakerError("breaker: exceeds consecutive fail times")
	ErrPercent       = NewBreakerError("breaker: exceeds error percent")
	ErrOpen          = NewBreakerError("breaker: initiative open")

	errDefault = NewBreakerError("breaker: error is nil")
)

// CheckerFunc is a function called by a Breaker's and check whether exceeds the
// breaker resource. It will receive the Breaker as an argument and returns an
// error. By default, a Breaker has no CheckerFunc.
type CheckerFunc func(*Breaker) error

// Breaker is the base of a circuit breaker. It maintains failure and success counters
// as well as the event subscribers.
type Breaker struct {
	// BackOff is the backoff policy that is used when determining if the breaker should
	// attempt to retry. A breaker created with NewBreaker will use an exponential backoff
	// policy by default.
	BackOff backoff.BackOff

	// ShouldTrip is a TripFunc that determines whether a Fail() call should trip the breaker.
	ShouldTripFail    []CheckerFunc
	ShouldTripSuccess []CheckerFunc

	// TODO
	Checkers []CheckerFunc

	// Clock is used for controlling time in tests.
	Clock clock.Clock

	// TODO
	Sys System

	_              [4]byte // pad to fix golang issue #599
	consecFailures int64
	lastFailure    int64 // stored as nanoseconds since the Unix epoch
	halfOpens      int64
	counts         *window
	nextBackOff    time.Duration
	tripped        int32
	trippedError   atomic.Value
	broken         int32
	eventReceivers []chan BreakerEvent
	listeners      []chan ListenerEvent
	backoffLock    sync.Mutex
	callCounts     int64
}

// Options holds breaker configuration options.
type Options struct {
	BackOff       backoff.BackOff
	Clock         clock.Clock
	WindowTime    time.Duration
	WindowBuckets int
	Name          string
	Sys           System
}

// NewBreakerWithOptions creates a base breaker with a specified backoff, clock and TripFunc
func NewBreakerWithOptions(options *Options) *Breaker {
	if options == nil {
		options = &Options{}
	}

	if options.Clock == nil {
		options.Clock = clock.New()
	}

	if options.Sys == nil {
		options.Sys = defaultSystem{}
	}

	if options.BackOff == nil {
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = defaultInitialBackOffInterval
		b.MaxElapsedTime = defaultBackoffMaxElapsedTime
		b.Clock = options.Clock
		b.Reset()
		options.BackOff = b
	}

	if options.WindowTime == 0 {
		options.WindowTime = DefaultWindowTime
	}

	if options.WindowBuckets == 0 {
		options.WindowBuckets = DefaultWindowBuckets
	}

	var checker []CheckerFunc
	checker = append(checker, MaxConcurrentCheckerFunc(options.Name))
	checker = append(checker, SystemLoadCheckerFunc(options.Name))
	checker = append(checker, QPSCheckerFunc(options.Name))

	var tripsFail []CheckerFunc
	//trips = append(trips, ThresholdTripFunc(options.Name))
	tripsFail = append(tripsFail, ConsecutiveTripFunc(options.Name))
	tripsFail = append(tripsFail, ErrorPercentTripFunc(options.Name))

	var tripsSuccess []CheckerFunc
	tripsSuccess = append(tripsSuccess, AverageRTTripFunc(options.Name))

	return &Breaker{
		BackOff:           options.BackOff,
		Clock:             options.Clock,
		Sys:               options.Sys,
		ShouldTripFail:    tripsFail,
		ShouldTripSuccess: tripsSuccess,
		nextBackOff:       options.BackOff.NextBackOff(),
		counts:            newWindow(options.WindowTime, options.WindowBuckets),
		Checkers:          checker,
	}
}

// Subscribe returns a channel of BreakerEvents. Whenever the breaker changes state,
// the state will be sent over the channel. See BreakerEvent for the types of events.
func (cb *Breaker) Subscribe() <-chan BreakerEvent {
	eventReader := make(chan BreakerEvent)
	output := make(chan BreakerEvent, 100)

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
	cb.eventReceivers = append(cb.eventReceivers, eventReader)
	return output
}

// AddListener adds a channel of ListenerEvents on behalf of a listener.
// The listener channel must be buffered.
func (cb *Breaker) AddListener(listener chan ListenerEvent) {
	cb.listeners = append(cb.listeners, listener)
}

// RemoveListener removes a channel previously added via AddListener.
// Once removed, the channel will no longer receive ListenerEvents.
// Returns true if the listener was found and removed.
func (cb *Breaker) RemoveListener(listener chan ListenerEvent) bool {
	for i, receiver := range cb.listeners {
		if listener == receiver {
			cb.listeners = append(cb.listeners[:i], cb.listeners[i+1:]...)
			return true
		}
	}
	return false
}

// Trip will trip the circuit breaker. After Trip() is called, Tripped() will
// return true.
func (cb *Breaker) Trip(err error) {
	atomic.StoreInt32(&cb.tripped, 1)
	now := cb.Clock.Now()
	atomic.StoreInt64(&cb.lastFailure, now.UnixNano())
	cb.sendEvent(BreakerTripped)
	if err == nil {
		cb.trippedError.Store(errDefault)
	} else {
		cb.trippedError.Store(err)
	}
}

// Reset will reset the circuit breaker. After Reset() is called, Tripped() will
// return false.
func (cb *Breaker) Reset() {
	atomic.StoreInt32(&cb.broken, 0)
	atomic.StoreInt32(&cb.tripped, 0)
	atomic.StoreInt64(&cb.halfOpens, 0)
	cb.ResetCounters()
	cb.sendEvent(BreakerReset)
}

// ResetCounters will reset only the failures, consecFailures, and success counters
func (cb *Breaker) ResetCounters() {
	atomic.StoreInt64(&cb.consecFailures, 0)
	cb.counts.Reset()
}

// Tripped returns true if the circuit breaker is tripped, false if it is reset.
func (cb *Breaker) Tripped() bool {
	return atomic.LoadInt32(&cb.tripped) == 1
}

// Break trips the circuit breaker and prevents it from auto resetting. Use this when
// manual control over the circuit breaker state is needed.
func (cb *Breaker) Break() {
	atomic.StoreInt32(&cb.broken, 1)
	cb.Trip(ErrOpen)
}

// Failures returns the number of failures for this circuit breaker.
func (cb *Breaker) Failures() int64 {
	return cb.counts.Failures()
}

// ConsecFailures returns the number of consecutive failures that have occured.
func (cb *Breaker) ConsecFailures() int64 {
	return atomic.LoadInt64(&cb.consecFailures)
}

// Successes returns the number of successes for this circuit breaker.
func (cb *Breaker) Successes() int64 {
	return cb.counts.Successes()
}

// Fail is used to indicate a failure condition the Breaker should record. It will
// increment the failure counters and store the time of the last failure. If the
// breaker has a TripFunc it will be called, tripping the breaker if necessary.
func (cb *Breaker) Fail() {
	cb.counts.Fail()
	atomic.AddInt64(&cb.consecFailures, 1)
	now := cb.Clock.Now()
	atomic.StoreInt64(&cb.lastFailure, now.UnixNano())
	cb.sendEvent(BreakerFail)
	for _, call := range cb.ShouldTripFail {
		if err := call(cb); err != nil {
			cb.Trip(err)
			return
		}
	}
}

// Success is used to indicate a success condition the Breaker should record. If
// the success was triggered by a retry attempt, the breaker will be Reset().
func (cb *Breaker) Success() {
	atomic.StoreInt64(&cb.consecFailures, 0)
	cb.counts.Success()
	for _, call := range cb.ShouldTripSuccess {
		if err := call(cb); err != nil {
			cb.Trip(err)
			return
		}
	}

	cb.backoffLock.Lock()
	cb.BackOff.Reset()
	cb.nextBackOff = cb.BackOff.NextBackOff()
	cb.backoffLock.Unlock()

	state := cb.state()
	if state == halfopen {
		cb.Reset()
	}
}

func (cb *Breaker) SuccessWithUpdate(r metricResult) {
	atomic.StoreInt64(&cb.consecFailures, 0)
	cb.counts.SuccessWithUpdate(r)
	for _, call := range cb.ShouldTripSuccess {
		if err := call(cb); err != nil {
			cb.Trip(err)
			return
		}
	}

	cb.backoffLock.Lock()
	cb.BackOff.Reset()
	cb.nextBackOff = cb.BackOff.NextBackOff()
	cb.backoffLock.Unlock()

	state := cb.state()
	if state == halfopen {
		cb.Reset()
	}
}

// ErrorRate returns the current error rate of the Breaker, expressed as a floating
// point number (e.g. 0.9 for 90%), since the last time the breaker was Reset.
func (cb *Breaker) ErrorRate() float64 {
	return cb.counts.ErrorRate()
}

// Ready will return true if the circuit breaker is ready to call the function.
// It will be ready if the breaker is in a reset state, or if it is time to retry
// the call for auto resetting.
func (cb *Breaker) Ready() bool {
	state := cb.state()
	if state == halfopen {
		atomic.StoreInt64(&cb.halfOpens, 0)
		cb.sendEvent(BreakerReady)
	}
	return state == closed || state == halfopen
}

func (cb *Breaker) Call(circuit func() error) error {
	var err error
	var start = time.Now()

	if !cb.Ready() {
		return cb.trippedError.Load().(error)
	}

	atomic.AddInt64(&cb.callCounts, 1)
	defer atomic.AddInt64(&cb.callCounts, -1)

	for _, ckr := range cb.Checkers {
		if err := ckr(cb); err != nil {
			return err
		}
	}
	err = circuit()
	if err != nil {
		if err != context.Canceled {
			cb.Fail()
		}
		return err
	}
	cb.SuccessWithUpdate(metricResult{time.Since(start)})
	return nil
}

// CallContext is same as Call but if the ctx is canceled after the circuit returned an error,
// the error will not be marked as a failure because the call was canceled intentionally.
func (cb *Breaker) CallContext(ctx context.Context, circuit func() error) error {
	var err error
	var start = time.Now()

	if !cb.Ready() {
		return cb.trippedError.Load().(error)
	}

	atomic.AddInt64(&cb.callCounts, 1)
	defer atomic.AddInt64(&cb.callCounts, -1)

	for _, ckr := range cb.Checkers {
		if err := ckr(cb); err != nil {
			return err
		}
	}

	c := make(chan error, 1)
	go func() {
		c <- circuit()
		close(c)
	}()

	select {
	case e := <-c:
		err = e
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		if ctx.Err() != context.Canceled {
			cb.Fail()
		}
		return err
	}
	cb.SuccessWithUpdate(metricResult{time.Since(start)})
	return nil
}

// state returns the state of the TrippableBreaker. The states available are:
// closed - the circuit is in a reset state and is operational
// open - the circuit is in a tripped state
// halfopen - the circuit is in a tripped state but the reset timeout has passed
func (cb *Breaker) state() state {
	tripped := cb.Tripped()
	if tripped {
		if atomic.LoadInt32(&cb.broken) == 1 {
			return open
		}

		last := atomic.LoadInt64(&cb.lastFailure)
		since := cb.Clock.Now().Sub(time.Unix(0, last))

		cb.backoffLock.Lock()
		defer cb.backoffLock.Unlock()

		if cb.nextBackOff != backoff.Stop && since > cb.nextBackOff {
			if atomic.CompareAndSwapInt64(&cb.halfOpens, 0, 1) {
				cb.nextBackOff = cb.BackOff.NextBackOff()
				return halfopen
			}
			return open
		}
		return open
	}
	return closed
}

func (cb *Breaker) sendEvent(event BreakerEvent) {
	for _, receiver := range cb.eventReceivers {
		receiver <- event
	}
	for _, listener := range cb.listeners {
		le := ListenerEvent{CB: cb, Event: event}
		select {
		case listener <- le:
		default:
			<-listener
			listener <- le
		}
	}
}
