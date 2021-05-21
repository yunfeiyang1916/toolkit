package tracing

import (
	"net/http/httptrace"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// Tracer holds tracing details for one HTTP request.
type requestTracer struct {
	// root opentracing.Span
	sp opentracing.Span
}

func (r *requestTracer) clientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn:              r.getConn,
		GotConn:              r.gotConn,
		PutIdleConn:          r.putIdleConn,
		GotFirstResponseByte: r.gotFirstResponseByte,
		Got100Continue:       r.got100Continue,
		DNSStart:             r.dnsStart,
		DNSDone:              r.dnsDone,
		ConnectStart:         r.connectStart,
		ConnectDone:          r.connectDone,
		WroteHeaders:         r.wroteHeaders,
		Wait100Continue:      r.wait100Continue,
		WroteRequest:         r.wroteRequest,
	}
}

func (r *requestTracer) getConn(hostPort string) {
	ext.PeerAddress.Set(r.sp, hostPort)
	r.sp.LogFields(log.String("event", "GetConn"))
}

func (r *requestTracer) gotConn(info httptrace.GotConnInfo) {
	r.sp.SetTag("net/http.reused", info.Reused)
	r.sp.SetTag("net/http.was_idle", info.WasIdle)
	r.sp.LogFields(log.String("event", "GotConn"))
}

func (r *requestTracer) putIdleConn(error) {
	r.sp.LogFields(log.String("event", "PutIdleConn"))
}

func (r *requestTracer) gotFirstResponseByte() {
	r.sp.LogFields(log.String("event", "GotFirstResponseByte"))
}

func (r *requestTracer) got100Continue() {
	r.sp.LogFields(log.String("event", "Got100Continue"))
}

func (r *requestTracer) dnsStart(info httptrace.DNSStartInfo) {
	r.sp.LogFields(
		log.String("event", "DNSStart"),
		log.String("host", info.Host),
	)
}

func (r *requestTracer) dnsDone(info httptrace.DNSDoneInfo) {
	fields := []log.Field{log.String("event", "DNSDone")}
	for _, addr := range info.Addrs {
		fields = append(fields, log.String("addr", addr.String()))
	}
	if info.Err != nil {
		fields = append(fields, log.Error(info.Err))
	}
	r.sp.LogFields(fields...)
}

func (r *requestTracer) connectStart(network, addr string) {
	r.sp.LogFields(
		log.String("event", "ConnectStart"),
		log.String("network", network),
		log.String("addr", addr),
	)
}

func (r *requestTracer) connectDone(network, addr string, err error) {
	if err != nil {
		r.sp.LogFields(
			log.String("message", "ConnectDone"),
			log.String("network", network),
			log.String("addr", addr),
			log.String("event", "error"),
			log.Error(err),
		)
	} else {
		r.sp.LogFields(
			log.String("event", "ConnectDone"),
			log.String("network", network),
			log.String("addr", addr),
		)
	}
}

func (r *requestTracer) wroteHeaders() {
	r.sp.LogFields(log.String("event", "WroteHeaders"))
}

func (r *requestTracer) wait100Continue() {
	r.sp.LogFields(log.String("event", "Wait100Continue"))
}

func (r *requestTracer) wroteRequest(info httptrace.WroteRequestInfo) {
	if info.Err != nil {
		r.sp.LogFields(
			log.String("message", "WroteRequest"),
			log.String("event", "error"),
			log.Error(info.Err),
		)
		ext.Error.Set(r.sp, true)
	} else {
		r.sp.LogFields(log.String("event", "WroteRequest"))
	}
}
