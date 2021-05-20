package sql

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/sql/trace"
)

// Client继承了*gorm.DB的所有方法, 详细的使用方法请参考:
// http://gorm.io/docs/connecting_to_the_database.html
type Client struct {
	*gorm.DB
}

type Group struct {
	name string

	master  *Client
	replica []*Client
	next    uint64
	total   uint64
}

func parseConnAddress(address string) (string, int, int, int, error) {
	u, err := mysql.ParseDSN(address)
	if err != nil {
		return address, -1, -1, 0, err
	}
	q := u.Params
	idleQ, activeQ, lifetimeQ := q["max_idle"], q["max_active"], q["max_lifetime_sec"]
	maxIdle, _ := strconv.Atoi(idleQ)
	if maxIdle == 0 {
		maxIdle = 15
	}
	maxActive, _ := strconv.Atoi(activeQ)
	lifetime, _ := strconv.Atoi(lifetimeQ)
	if lifetime == 0 {
		lifetime = 1800
	}
	delete(q, "max_idle")
	delete(q, "max_active")
	delete(q, "max_lifetime_sec")
	return u.FormatDSN(), maxIdle, maxActive, lifetime, nil
}

func openDB(name, address string, isMaster int, statLevel, format string) (*Client, error) {
	addr, maxIdle, maxActive, lifetime, err := parseConnAddress(address)
	if err != nil {
		return nil, err
	}
	db, err := trace.Open(addr, name)
	if err != nil {
		return nil, fmt.Errorf("open mysql [%s] master %s error %s", name, address, err)
	}
	db = db.Debug()
	db = db.BlockGlobalUpdate(true)
	db.SetLogger(newGlobalLogger(statLevel, isMaster, parseDbName(address), format))
	db.DB().SetMaxIdleConns(maxIdle)
	db.DB().SetMaxOpenConns(maxActive)
	db.DB().SetConnMaxLifetime(time.Duration(lifetime) * time.Second)

	return &Client{DB: db}, err
}

// NewGroup 初始化一个Group， 一个Group包含一个master实例和零个或多个slave实例
func NewGroup(d SQLGroupConfig) (*Group, error) {
	log.Infof("init sql group name [%s], master [%s], slave [%v]", d.Name, d.Master, d.Slaves)
	g := Group{name: d.Name}
	var err error
	g.master, err = openDB(d.Name, d.Master, 1, d.StatLevel, d.LogFormat)
	if err != nil {
		return nil, err
	}
	g.replica = make([]*Client, 0, len(d.Slaves))
	g.total = 0
	for _, slave := range d.Slaves {
		c, err := openDB(d.Name, slave, 0, d.StatLevel, d.LogFormat)
		if err != nil {
			return nil, err
		}
		g.replica = append(g.replica, c)
		g.total++

	}
	return &g, nil
}

// Master 返回master实例
func (g *Group) Master() *Client {
	return g.master
}

// Slave 返回一个slave实例，使用轮转算法
func (g *Group) Slave() *Client {
	if g.total == 0 {
		return g.master
	}
	next := atomic.AddUint64(&g.next, 1)
	return g.replica[next%g.total]
}

// Instance 函数如果isMaster是true， 返回master实例，否则返回slave实例
func (g *Group) Instance(isMaster bool) *Client {
	if isMaster {
		return g.Master()
	}
	return g.Slave()
}

func parseDbName(s string) string {
	u, err := mysql.ParseDSN(s)
	if err != nil {
		return s
	}
	return u.DBName
}
