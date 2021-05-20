// +build go1.10

package trace

import "database/sql"

func OpenDB(dsn, name string) *sql.DB {
	tc := &tracedConnector{
		dsn:      dsn,
		instance: name,
	}
	return sql.OpenDB(tc)
}
