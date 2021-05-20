package trace

import (
	"context"
	"database/sql/driver"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracinglog "github.com/opentracing/opentracing-go/log"
	"github.com/yunfeiyang1916/toolkit/go-tls"
)

var _ driver.Conn = (*tracedConn)(nil)

type tracedConn struct {
	driver.Conn
	*traceParams
}

func (tc *tracedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	start := time.Now()
	if connBeginTx, ok := tc.Conn.(driver.ConnBeginTx); ok {
		tx, err = connBeginTx.BeginTx(ctx, opts)
		tc.tryTrace(ctx, "Begin", "", start, nil, err)
		if err != nil {
			return nil, err
		}
		return &tracedTx{tx, tc.traceParams, ctx}, nil
	}
	tx, err = tc.Conn.Begin()
	tc.tryTrace(ctx, "Begin", "", start, nil, err)
	if err != nil {
		return nil, err
	}
	return &tracedTx{tx, tc.traceParams, ctx}, nil
}

func (tc *tracedConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	start := time.Now()
	if connPrepareCtx, ok := tc.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := connPrepareCtx.PrepareContext(ctx, query)
		tc.tryTrace(ctx, "Prepare", query, start, nil, err)
		if err != nil {
			return nil, err
		}
		return &tracedStmt{stmt, tc.traceParams, ctx, query}, nil
	}
	stmt, err = tc.Prepare(query)
	tc.tryTrace(ctx, "Prepare", query, start, nil, err)
	if err != nil {
		return nil, err
	}
	return &tracedStmt{stmt, tc.traceParams, ctx, query}, nil
}

func (tc *tracedConn) Exec(query string, args []driver.Value) (res driver.Result, err error) {
	if execer, ok := tc.Conn.(driver.Execer); ok {
		return execer.Exec(query, args)
	}
	return nil, driver.ErrSkip
}

func (tc *tracedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (r driver.Result, err error) {
	start := time.Now()
	if execContext, ok := tc.Conn.(driver.ExecerContext); ok {
		r, err := execContext.ExecContext(ctx, query, args)
		tc.tryTrace(ctx, "Exec", query, start, args, err)
		return r, err
	}
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	r, err = tc.Exec(query, dargs)
	tc.tryTrace(ctx, "Exec", query, start, args, err)
	return r, err
}

// tracedConn has a Ping method in order to implement the pinger interface
func (tc *tracedConn) Ping(ctx context.Context) (err error) {
	start := time.Now()
	if pinger, ok := tc.Conn.(driver.Pinger); ok {
		err = pinger.Ping(ctx)
	}
	tc.tryTrace(ctx, "Ping", "", start, nil, err)
	return err
}

func (tc *tracedConn) Query(query string, args []driver.Value) (row driver.Rows, err error) {
	if queryer, ok := tc.Conn.(driver.Queryer); ok {
		return queryer.Query(query, args)
	}
	return nil, driver.ErrSkip
}

func (tc *tracedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	start := time.Now()
	if queryerContext, ok := tc.Conn.(driver.QueryerContext); ok {
		rows, err = queryerContext.QueryContext(ctx, query, args)
		tc.tryTrace(ctx, "Query", query, start, args, err)
		return rows, err
	}
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	rows, err = tc.Query(query, dargs)
	tc.tryTrace(ctx, "Query", query, start, args, err)
	return rows, err
}

// traceParams stores all information related to tracing the driver.Conn
type traceParams struct {
	host     string
	user     string
	database string
	instance string
}

// tryTrace will create a span using the given arguments, but will act as a no-op when err is driver.ErrSkip.
func (tp *traceParams) tryTrace(ctx context.Context, resource string, query string, startTime time.Time, nargs []driver.NamedValue, err error) {
	if err == driver.ErrSkip {
		return
	}
	if ctx == context.Background() || ctx == context.TODO() {
		myctx, ok := tls.GetContext()
		if !ok {
			return
		}

		ctx = myctx
	}
	span, _ := opentracing.StartSpanFromContext(ctx, "MYSQL Client "+resource, opentracing.StartTime(startTime))
	// annotation
	ext.PeerAddress.Set(span, tp.host)
	ext.DBType.Set(span, "database/mysql")
	ext.Component.Set(span, "golang/mysql-client")
	ext.DBUser.Set(span, tp.user)
	ext.DBInstance.Set(span, tp.instance)
	if resource == "Prepare" {
		ext.DBStatement.Set(span, query)
	}
	if len(query) != 0 || len(nargs) != 0 {
		span.LogFields(opentracinglog.String("Query", query), opentracinglog.Object("NamedValue", nargs))
	}
	if err != nil {
		ext.Error.Set(span, true)
		span.LogFields(opentracinglog.Error(err))
	}
	span.Finish()
}
