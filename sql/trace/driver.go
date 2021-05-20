package trace

import (
	"context"
	"database/sql/driver"

	"github.com/go-sql-driver/mysql"
)

var mysqlDriver = &mysql.MySQLDriver{}

type tracedConnector struct {
	dsn      string
	instance string
}

func (t *tracedConnector) Connect(c context.Context) (driver.Conn, error) {
	conn, err := mysqlDriver.Open(t.dsn)
	if err != nil {
		return nil, err
	}
	tp := &traceParams{}
	q, _ := mysql.ParseDSN(t.dsn)
	if q != nil {
		tp.host = q.Addr
		tp.user = q.User
		tp.database = q.DBName
		tp.instance = t.instance
	}
	return &tracedConn{conn, tp}, err
}

func (t *tracedConnector) Driver() driver.Driver {
	return mysqlDriver
}

type tracedDriver struct {
}

func (*tracedDriver) Open(name string) (driver.Conn, error) {
	conn, err := mysqlDriver.Open(name)
	if err != nil {
		return nil, err
	}
	tp := &traceParams{}
	q, _ := mysql.ParseDSN(name)
	if q != nil {
		tp.host = q.Addr
		tp.user = q.User
		tp.database = q.DBName
	}
	return &tracedConn{conn, tp}, err
}
