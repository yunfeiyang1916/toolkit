package circuit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenk/backoff"
	"github.com/facebookgo/clock"
)

const breakerName = "test"

func init() {
	defaultInitialBackOffInterval = time.Millisecond
}

// NewBreaker creates a base breaker with an exponential backoff and no TripFunc
func NewBreaker() *Breaker {
	return NewBreakerWithOptions(nil)
}

// NewThresholdBreaker creates a Breaker with a ThresholdTripFunc.
func NewThresholdBreaker(name string) *Breaker {
	return NewBreakerWithOptions(&Options{Name: name})
}

// NewConsecutiveBreaker creates a Breaker with a ConsecutiveTripFunc.
func NewConsecutiveBreaker(name string) *Breaker {
	return NewBreakerWithOptions(&Options{Name: name})
}

// NewRateBreaker creates a Breaker with a RateTripFunc.
func NewRateBreaker(name string) *Breaker {
	return NewBreakerWithOptions(&Options{Name: name})
}

func TestBreakerTripping(t *testing.T) {
	cb := NewBreaker()

	if cb.Tripped() {
		t.Fatal("expected breaker to not be tripped")
	}

	cb.Trip(nil)
	if !cb.Tripped() {
		t.Fatal("expected breaker to be tripped")
	}

	cb.Reset()
	if cb.Tripped() {
		t.Fatal("expected breaker to have been reset")
	}
}

func TestBreakerCounts(t *testing.T) {
	cb := NewBreaker()

	cb.Fail()
	if failures := cb.Failures(); failures != 1 {
		t.Fatalf("expected failure count to be 1, got %d", failures)
	}

	cb.Fail()
	if consecFailures := cb.ConsecFailures(); consecFailures != 2 {
		t.Fatalf("expected 2 consecutive failures, got %d", consecFailures)
	}

	cb.Success()
	if successes := cb.Successes(); successes != 1 {
		t.Fatalf("expected success count to be 1, got %d", successes)
	}
	if consecFailures := cb.ConsecFailures(); consecFailures != 0 {
		t.Fatalf("expected 0 consecutive failures, got %d", consecFailures)
	}

	cb.Reset()
	if failures := cb.Failures(); failures != 0 {
		t.Fatalf("expected failure count to be 0, got %d", failures)
	}
	if successes := cb.Successes(); successes != 0 {
		t.Fatalf("expected success count to be 0, got %d", successes)
	}
	if consecFailures := cb.ConsecFailures(); consecFailures != 0 {
		t.Fatalf("expected 0 consecutive failures, got %d", consecFailures)
	}
}

func TestErrorRate(t *testing.T) {
	cb := NewBreaker()
	if er := cb.ErrorRate(); er != 0.0 {
		t.Fatalf("expected breaker with no samples to have 0 error rate, got %f", er)
	}
}

func TestBreakerEvents(t *testing.T) {
	c := clock.NewMock()
	cb := NewBreaker()
	cb.Clock = c
	events := cb.Subscribe()

	cb.Trip(nil)
	if e := <-events; e != BreakerTripped {
		t.Fatalf("expected to receive a trip event, got %d", e)
	}

	c.Add(cb.nextBackOff + 1)
	cb.Ready()
	if e := <-events; e != BreakerReady {
		t.Fatalf("expected to receive a breaker ready event, got %d", e)
	}

	cb.Reset()
	if e := <-events; e != BreakerReset {
		t.Fatalf("expected to receive a reset event, got %d", e)
	}

	cb.Fail()
	if e := <-events; e != BreakerFail {
		t.Fatalf("expected to receive a fail event, got %d", e)
	}
}

func TestAddRemoveListener(t *testing.T) {
	c := clock.NewMock()
	cb := NewBreaker()
	cb.Clock = c
	events := make(chan ListenerEvent, 100)
	cb.AddListener(events)

	cb.Trip(nil)
	if e := <-events; e.Event != BreakerTripped {
		t.Fatalf("expected to receive a trip event, got %v", e)
	}

	c.Add(cb.nextBackOff + 1)
	cb.Ready()
	if e := <-events; e.Event != BreakerReady {
		t.Fatalf("expected to receive a breaker ready event, got %v", e)
	}

	cb.Reset()
	if e := <-events; e.Event != BreakerReset {
		t.Fatalf("expected to receive a reset event, got %v", e)
	}

	cb.Fail()
	if e := <-events; e.Event != BreakerFail {
		t.Fatalf("expected to receive a fail event, got %v", e)
	}

	cb.RemoveListener(events)
	cb.Reset()
	select {
	case e := <-events:
		t.Fatalf("after removing listener, should not receive reset event; got %v", e)
	default:
		// Expected.
	}
}

func TestTrippableBreakerState(t *testing.T) {
	c := clock.NewMock()
	cb := NewBreaker()
	cb.Clock = c

	if !cb.Ready() {
		t.Fatal("expected breaker to be ready")
	}

	cb.Trip(nil)
	if cb.Ready() {
		t.Fatal("expected breaker to not be ready")
	}
	c.Add(cb.nextBackOff + 1)
	if !cb.Ready() {
		t.Fatal("expected breaker to be ready after reset timeout")
	}

	cb.Fail()
	c.Add(cb.nextBackOff + 1)
	if !cb.Ready() {
		t.Fatal("expected breaker to be ready after reset timeout, post failure")
	}
}

func TestTrippableBreakerManualBreak(t *testing.T) {
	c := clock.NewMock()
	cb := NewBreaker()
	cb.Clock = c
	cb.Break()
	c.Add(cb.nextBackOff + 1)

	if cb.Ready() {
		t.Fatal("expected breaker to still be tripped")
	}

	cb.Reset()
	cb.Trip(nil)
	c.Add(cb.nextBackOff + 1)
	if !cb.Ready() {
		t.Fatal("expected breaker to be ready")
	}
}

/*
func TestThresholdBreaker(t *testing.T) {
	cb := NewThresholdBreaker(breakerName)

	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}

	cb.Fail()
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to still be open")
	}

	cb.Fail()
	if !cb.Tripped() {
		t.Fatal("expected threshold breaker to be tripped")
	}

	cb.Reset()
	if failures := cb.Failures(); failures != 0 {
		t.Fatalf("expected reset to set failures to 0, got %d", failures)
	}
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}
}
*/

func TestConsecutiveBreaker(t *testing.T) {
	SettingConsecutiveError(breakerName, true, 3)
	cb := NewConsecutiveBreaker(breakerName)

	if cb.Tripped() {
		t.Fatal("expected consecutive breaker to be open")
	}

	cb.Fail()
	cb.Success()
	cb.Fail()
	cb.Fail()
	if cb.Tripped() {
		t.Fatal("expected consecutive breaker to be open")
	}
	cb.Fail()
	if !cb.Tripped() {
		t.Fatal("expected consecutive breaker to be tripped")
	}
	cb.Fail()
	cb.Fail()
	cb.Fail()
	time.Sleep(time.Millisecond * 10)
	if !cb.Tripped() {
		t.Fatal("expected consecutive breaker to be tripped")
	}
	time.Sleep(cb.nextBackOff)
	cb.Success()

	if cb.Tripped() {
		t.Fatal("expected consecutive breaker to be open")
	}
}

/*
func TestThresholdBreakerCallContexting(t *testing.T) {
	circuit := func() error {
		return fmt.Errorf("error")
	}

	cb := NewThresholdBreaker(breakerName)

	err := cb.CallContext(circuit, 0) // First failure
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}

	err = cb.CallContext(circuit, 0) // Second failure trips
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if !cb.Tripped() {
		t.Fatal("expected threshold breaker to be tripped")
	}
}

func TestThresholdBreakerCallContextingContext(t *testing.T) {
	circuit := func() error {
		return fmt.Errorf("error")
	}

	cb := NewThresholdBreaker(breakerName)
	ctx, cancel := context.WithCancel(context.Background())

	err := cb.CallContextContext(ctx, circuit, 0) // First failure
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}

	// Cancel the next CallContext.
	cancel()

	err = cb.CallContextContext(ctx, circuit, 0) // Second failure but it's canceled
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}

	err = cb.CallContextContext(context.Background(), circuit, 0) // Thirt failure trips
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if !cb.Tripped() {
		t.Fatal("expected threshold breaker to be tripped")
	}
}

func TestThresholdBreakerResets(t *testing.T) {
	called := 0
	success := false
	circuit := func() error {
		if called == 0 {
			called++
			return fmt.Errorf("error")
		}
		success = true
		return nil
	}

	c := clock.NewMock()
	cb := NewThresholdBreaker(breakerName)
	cb.Clock = c
	err := cb.CallContext(circuit, 0)
	if err == nil {
		t.Fatal("Expected cb to return an error")
	}

	c.Add(cb.nextBackOff + 1)
	for i := 0; i < 4; i++ {
		err = cb.CallContext(circuit, 0)
		if err != nil {
			t.Fatal("Expected cb to be successful")
		}

		if !success {
			t.Fatal("Expected cb to have been reset")
		}
	}
}
*/

func TestCancel(t *testing.T) {
	circuit := func() error {
		time.Sleep(time.Second)
		return nil
	}

	errc := make(chan error)
	cb := NewConsecutiveBreaker("TestCancel")
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2000)
	go func() { errc <- cb.CallContext(ctx, circuit) }()

	go func() {
		time.Sleep(time.Millisecond * 200)
		cancel()
	}()

	err := <-errc
	if err != context.Canceled {
		t.Fatal("unexpected error", err)
	}
}

func TestTimeoutBreaker(t *testing.T) {
	wait := make(chan struct{})

	c := clock.NewMock()
	called := int32(0)

	circuit := func() error {
		wait <- struct{}{}
		atomic.AddInt32(&called, 1)
		<-wait
		return nil
	}

	SettingConsecutiveError("TestTimeoutBreaker", true, 2)
	cb := NewConsecutiveBreaker("TestTimeoutBreaker")
	cb.Clock = c

	errc := make(chan error)
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	go func() { errc <- cb.CallContext(ctx, circuit) }()
	<-wait

	time.Sleep(time.Millisecond * 3)
	wait <- struct{}{}

	err := <-errc
	if err != context.DeadlineExceeded {
		t.Fatal("expected timeout breaker to return an error")
	}

	if cb.Tripped() {
		t.Fatal("expected timeout breaker to be close")
	}

	ctx, _ = context.WithTimeout(context.Background(), time.Millisecond)
	go cb.CallContext(ctx, circuit)
	<-wait
	time.Sleep(time.Millisecond * 3)
	wait <- struct{}{}

	if !cb.Tripped() {
		t.Fatal("expected timeout breaker to be open")
	}
}

func TestRateBreakerTripping(t *testing.T) {
	SettingErrorPercent("TestRateBreakerTripping", true, 50, 4)
	cb := NewRateBreaker("TestRateBreakerTripping")
	cb.Success()
	cb.Success()
	cb.Fail()
	cb.Fail()

	if !cb.Tripped() {
		t.Fatal("expected rate breaker to be tripped")
	}

	if er := cb.ErrorRate(); er != 0.5 {
		t.Fatalf("expected error rate to be 0.5, got %f", er)
	}
}

func TestRateBreakerSampleSize(t *testing.T) {
	cb := NewRateBreaker(breakerName)
	cb.Fail()

	if cb.Tripped() {
		t.Fatal("expected rate breaker to not be tripped yet")
	}
}

func TestRateBreakerResets(t *testing.T) {
	serviceError := fmt.Errorf("service error")

	called := 0
	success := false
	circuit := func() error {
		if called < 4 {
			called++
			return serviceError
		}
		success = true
		return nil
	}

	c := clock.NewMock()
	SettingErrorPercent("TestRateBreakerResets", true, 50, 4)
	cb := NewRateBreaker("TestRateBreakerResets")
	cb.Clock = c
	var err error
	for i := 0; i < 4; i++ {
		err = cb.CallContext(context.Background(), circuit)
		if err == nil {
			t.Fatal("Expected cb to return an error (closed breaker, service failure)")
		} else if err != serviceError {
			t.Fatal("Expected cb to return error from service (closed breaker, service failure)")
		}
	}

	err = cb.CallContext(context.Background(), circuit)
	if err == nil {
		t.Fatal("Expected cb to return an error (open breaker)")
	} else if err != ErrPercent {
		t.Fatal("Expected cb to return open open breaker error (open breaker)", err)
	}

	c.Add(cb.nextBackOff + 1)
	err = cb.CallContext(context.Background(), circuit)
	if err != nil {
		t.Fatal("Expected cb to be successful")
	}

	if !success {
		t.Fatal("Expected cb to have been reset")
	}
}

func TestNeverRetryAfterBackoffStops(t *testing.T) {
	cb := NewBreakerWithOptions(&Options{
		BackOff: &backoff.StopBackOff{},
	})

	cb.Trip(nil)

	// circuit should be open and never retry again
	// when nextBackoff is backoff.Stop
	called := 0
	cb.CallContext(context.Background(), func() error {
		called = 1
		return nil
	})

	if called == 1 {
		t.Fatal("Expected cb to never retry")
	}
}

// TestPartialSecondBackoff ensures that the breaker event less than nextBackoff value
// time after tripping the breaker isn't allowed.
func TestPartialSecondBackoff(t *testing.T) {
	c := clock.NewMock()
	cb := NewBreaker()
	cb.Clock = c

	// Set the time to 0.5 seconds after the epoch, then trip the breaker.
	c.Add(500 * time.Millisecond)
	cb.Trip(nil)

	// Move forward 100 milliseconds in time and ensure that the backoff time
	// is set to a larger number than the clock advanced.
	c.Add(100 * time.Millisecond)
	cb.nextBackOff = 500 * time.Millisecond
	if cb.Ready() {
		t.Fatalf("expected breaker not to be ready after less time than nextBackoff had passed")
	}

	c.Add(401 * time.Millisecond)
	if !cb.Ready() {
		t.Fatalf("expected breaker to be ready after more than nextBackoff time had passed")
	}
}

func TestRTTrip(t *testing.T) {
	SettingAverageRT("TestRTTrip", true, time.Millisecond*7)
	cb := NewBreakerWithOptions(&Options{Name: "TestRTTrip"})
	circuit := func() error {
		time.Sleep(time.Millisecond * 8)
		return nil
	}

	wait := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			cb.Call(circuit)
		}()
	}
	wait.Wait()
	if !cb.Tripped() {
		t.Fatal("expected rt breaker to be open")
	}

	time.Sleep(cb.nextBackOff)
	cb.Success()
	if !cb.Tripped() {
		t.Fatal("expected rt breaker to be open")
	}

	circuit2 := func() error {
		return nil
	}

	err := cb.Call(circuit2)
	if err != ErrAverageRT || !cb.Tripped() {
		t.Fatal("expected breaker open")
	}

	for i := 0; i < 10; i++ {
		time.Sleep(cb.nextBackOff)
		cb.Call(circuit2)
		mean := time.Duration(cb.counts.Metric().Mean())
		if mean > time.Millisecond*7 && !cb.Tripped() {
			t.Fatal("expected breaker open", err, mean)
		}
		if mean <= time.Millisecond*7 && cb.Tripped() {
			t.Fatal("expected breaker close", err, mean)
		}
	}
}

// TestRTTrip ensures RT breaker is working.
func TestRTTripContext(t *testing.T) {
	SettingAverageRT("TestRTTripContext", true, time.Millisecond*7)
	cb := NewBreakerWithOptions(&Options{Name: "TestRTTripContext"})
	circuit := func() error {
		time.Sleep(time.Millisecond * 8)
		return nil
	}

	wait := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			cb.CallContext(context.Background(), circuit)
		}()
	}
	wait.Wait()
	if !cb.Tripped() {
		t.Fatal("expected rt breaker to be open")
	}

	time.Sleep(cb.nextBackOff)
	cb.Success()
	if !cb.Tripped() {
		t.Fatal("expected rt breaker to be open")
	}

	circuit2 := func() error {
		return nil
	}

	err := cb.CallContext(context.Background(), circuit2)
	if err != ErrAverageRT || !cb.Tripped() {
		t.Fatal("expected breaker open")
	}

	for i := 0; i < 10; i++ {
		time.Sleep(cb.nextBackOff)
		cb.CallContext(context.Background(), circuit2)
		mean := time.Duration(cb.counts.Metric().Mean())
		if mean > time.Millisecond*7 && !cb.Tripped() {
			t.Fatal("expected breaker open", err, mean)
		}
		if mean <= time.Millisecond*7 && cb.Tripped() {
			t.Fatal("expected breaker close", err, mean)
		}
	}
}

func TestMaxConcurrentChecker(t *testing.T) {
	max := int32(10)
	SettingMaxConcurrent("TestMaxConcurrentChecker", true, int64(max))
	cb := NewBreakerWithOptions(&Options{Name: "TestMaxConcurrentChecker"})

	wait := make(chan struct{})
	exit := make(chan struct{})
	called := int32(0)
	circuit := func() error {
		atomic.AddInt32(&called, 1)
		<-wait
		return nil
	}

	for i := 0; i < int(max); i++ {
		go func() {
			cb.CallContext(context.Background(), circuit)
			exit <- struct{}{}
		}()
	}
	for atomic.LoadInt32(&called) != max {
		time.Sleep(time.Millisecond * 10)
	}
	if cb.Tripped() {
		t.Fatal("expected breaker to be close")
	}

	errc := make(chan error)
	go func() {
		errc <- cb.CallContext(context.Background(), circuit)
	}()

	err := <-errc
	if err != ErrMaxConcurrent {
		t.Fatal("expected an error")
	}

	if cb.Tripped() {
		t.Fatal("expected breaker to be open")
	}

	wait <- struct{}{}
	<-exit

	go func() {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
		errc <- cb.CallContext(ctx, circuit)
	}()

	err = <-errc
	if err != context.DeadlineExceeded {
		t.Fatal("expected no error", err)
	}
}

func TestQPSLimitReject(t *testing.T) {
	SettingQPSLimitReject("TestQPSLimitReject", true, 10)
	cb := NewBreakerWithOptions(&Options{Name: "TestQPSLimitReject"})

	circuit := func() error {
		return nil
	}

	for i := 0; i < 20; i++ {
		err := cb.CallContext(context.Background(), circuit)
		if err == ErrRateLimit && i < 10 {
			t.Fatal(err)
		}
		if err == nil && i > 10 {
			t.Fatal("unexpected")
		}
	}
	if err := cb.CallContext(context.Background(), circuit); err != ErrRateLimit {
		t.Fatal("expected ErrRateLimit error", err)
	}
	time.Sleep(time.Second * 2)
	if err := cb.CallContext(context.Background(), circuit); err != nil {
		t.Fatal("unexpected error", err)
	}
}

func TestQPSLimitLeakyBucket(t *testing.T) {
	SettingQPSLimitLeakyBucket("TestQPSLimitLeakyBucket", true, 10)
	cb := NewBreakerWithOptions(&Options{Name: "TestQPSLimitLeakyBucket"})
	errc := make(chan error, 20)
	count := int32(0)
	var i = new(atomic.Value)
	i.Store(time.Unix(0, 0))
	circuit := func() error {
		now := time.Now()
		tick := i.Load().(time.Time)
		if tick.Unix() != 0 && (now.Sub(tick) < 98*time.Millisecond || now.Sub(tick) > 102*time.Millisecond) {
			errc <- errors.New(now.Sub(tick).String())
		}
		i.Store(now)
		return nil
	}

	for i := 0; i < 20; i++ {
		go func() {
			errc <- cb.CallContext(context.Background(), circuit)
			atomic.AddInt32(&count, 1)
			if atomic.LoadInt32(&count) == 20 {
				close(errc)
			}
		}()
	}

	for err := range errc {
		if err != nil {
			t.Fatal("unexpected error", err)
		}
	}
}

func TestLeakyBucketWithMaxConcurrent(t *testing.T) {
	SettingQPSLimitLeakyBucket("TestLeakyBucketWithMaxConcurrent", true, 10)
	SettingMaxConcurrent("TestLeakyBucketWithMaxConcurrent", true, int64(20))
	cb := NewBreakerWithOptions(&Options{
		Name: "TestLeakyBucketWithMaxConcurrent",
	})

	errc := make(chan error, 10)
	circuit := func() error {
		return nil
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 31; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errc <- cb.CallContext(context.Background(), circuit)
		}()
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	maxerr := int32(0)
	for err := range errc {
		if err == ErrMaxConcurrent {
			atomic.AddInt32(&maxerr, 1)
			continue
		}
		if err != nil {
			t.Fatal("unexpected error", err)
		}
	}
	if c := atomic.LoadInt32(&maxerr); c != 10 {
		t.Fatal("unexpected behavior", c)
	}
}

func TestSyetemLoads(t *testing.T) {
	SettingSystemLoads("TestSyetemLoads", true, 9.0)
	//mock := &mockSystem{value: 10.0}
	//cb := NewBreakerWithOptions(&Options{
	//	Name: "TestSyetemLoads",
	//	Sys:  mock,
	//})
	//
	//circuit := func() error {
	//	return nil
	//}
	//for i := 0; i < 31; i++ {
	//	switch i {
	//	case 10:
	//		mock.Open()
	//	case 20:
	//		mock.Close()
	//	}
	//	err := cb.CallContext(context.Background(), circuit)
	//	if err == nil && (i > 10 && i < 20) {
	//		t.Fatal("unexpected behavior", i, atomic.LoadInt32(&mock.open))
	//	}
	//	if err != nil && err != ErrSystemLoad {
	//		t.Fatal("unexpected behavior")
	//	}
	//}
}

func TestAll(t *testing.T) {
	//	SettingQPSLimitLeakyBucket("TestAll", true, 10000)
	//	SettingMaxConcurrent("TestAll", true, int64(20))
	//	SettingErrorPercent("TestAll", true, 50, 100)
	return
	SettingAverageRT("TestAll", true, time.Millisecond*3)
	cb := NewBreakerWithOptions(&Options{Name: "TestAll"})

	//count := int32(0)
	circuit := func() error {
		time.Sleep(time.Millisecond * 100)
		return nil
	}

	wg := &sync.WaitGroup{}
	errc := make(chan error, 1000)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errc <- cb.CallContext(context.Background(), circuit)
		}()
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		t.Log(err)
	}
}
