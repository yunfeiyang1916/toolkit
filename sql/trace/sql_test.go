package trace

import (
	"fmt"
	"testing"
)

func TestOpenDB(t *testing.T) {
	master := "admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"
	db := OpenDB(master, "admin_test")
	fmt.Println(db.Ping())
}
