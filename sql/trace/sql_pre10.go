// +build !go1.10

package trace

import (
	"database/sql"
)

func OpenDB(dsn, name string) *sql.DB {
	d, _ := sql.Open("traced-mysql", dsn)
	return d
}

func init() {
	sql.Register("traced-mysql", &tracedDriver{})
}
