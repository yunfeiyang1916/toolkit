package log

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	goctx "golang.org/x/net/context"
)

// A Valuer generates a log value. When passed to With or WithPrefix in a
// value element (odd indexes), it represents a dynamic value which is re-
// evaluated with each log event.
type Valuer func() interface{}

// bindValues replaces all value elements (odd indexes) containing a Valuer
// with their generated value.
//nolint:unused
func bindValues(keyvals []interface{}) {
	for i := 1; i < len(keyvals); i += 2 {
		if v, ok := keyvals[i].(Valuer); ok {
			keyvals[i] = v()
		}
	}
}

// containsValuer returns true if any of the value elements (odd indexes)
// contain a Valuer.
//nolint:unused
func containsValuer(keyvals []interface{}) bool {
	for i := 1; i < len(keyvals); i += 2 {
		if _, ok := keyvals[i].(Valuer); ok {
			return true
		}
	}
	return false
}

// Timestamp returns a timestamp Valuer. It invokes the t function to get the
// time; unless you are doing something tricky, pass time.Now.
//
// Most users will want to use DefaultTimestamp or DefaultTimestampUTC, which
// are TimestampFormats that use the RFC3339Nano format.
func Timestamp(t func() time.Time) Valuer {
	return func() interface{} { return t() }
}

// TimestampFormat returns a timestamp Valuer with a custom time format. It
// invokes the t function to get the time to format; unless you are doing
// something tricky, pass time.Now. The layout string is passed to
// Time.Format.
//
// Most users will want to use DefaultTimestamp or DefaultTimestampUTC, which
// are TimestampFormats that use the RFC3339Nano format.
func TimestampFormat(t func() time.Time, layout string) Valuer {
	return func() interface{} {
		return timeFormat{
			time:   t(),
			layout: layout,
		}
	}
}

// A timeFormat represents an instant in time and a layout used when
// marshaling to a text format.
type timeFormat struct {
	time   time.Time
	layout string
}

func (tf timeFormat) String() string {
	return tf.time.Format(tf.layout)
}

// MarshalText implements encoding.TextMarshaller.
func (tf timeFormat) MarshalText() (text []byte, err error) {
	// The following code adapted from the standard library time.Time.Format
	// method. Using the same undocumented magic constant to extend the size
	// of the buffer as seen there.
	b := make([]byte, 0, len(tf.layout)+10)
	b = tf.time.AppendFormat(b, tf.layout)
	return b, nil
}

// Caller returns a Valuer that returns a file and line from a specified depth
// in the callstack. Users will probably want to use DefaultCaller.
func Caller(depth int) Valuer {
	return func() interface{} {
		_, file, line, _ := runtime.Caller(depth)
		idx := strings.LastIndexByte(file, '/')
		// using idx+1 below handles both of following cases:
		// idx == -1 because no "/" was found, or
		// idx >= 0 and we want to start at the character after the found "/".
		return file[idx+1:] + ":" + strconv.Itoa(line)
	}
}

// TraceID todo 使用jaeger，暂时注释
func TraceID(ctx goctx.Context) Valuer {
	// span := opentracing.SpanFromContext(ctx)
	return func() interface{} {
		var traceID string
		//if span, ok := span.(*jaeger.Span); ok {
		//	spanCtx := span.Context()
		//	if sc, ok := spanCtx.(jaeger.SpanContext); ok {
		//		traceID = sc.TraceID().String()
		//	}
		//}
		return traceID
	}
}

func Cost(t time.Time) Valuer {
	return func() interface{} {
		return fmt.Sprintf("%dms", time.Since(t).Nanoseconds()/1e6)
	}
}

var (
	// DefaultTimestamp is a Valuer that returns the current wallclock time,
	// respecting time zones, when bound.
	//DefaultTimestamp = TimestampFormat(time.Now, time.RFC3339)
	DefaultTimestamp = TimestampFormat(time.Now, "2006-01-02 15:04:05.999")

	// DefaultTimestampUTC is a Valuer that returns the current time in UTC
	// when bound.
	DefaultTimestampUTC = TimestampFormat(
		func() time.Time { return time.Now().UTC() },
		time.RFC3339Nano,
	)

	PID = func() interface{} {
		return strconv.Itoa(os.Getpid())
	}

	// DefaultCaller is a Valuer that returns the file and line where the Log
	// method was invoked. It can only be used with log.With.
	DefaultCaller = Caller(3)
)
