package sql

import (
	"sync"
)

// GroupManager 提供了Add和Get操作， 用于管理Group
type GroupManager struct {
	mu     sync.RWMutex
	groups map[string]*Group
}

var (
	// SQLGroupManager 是GroupManager结构体的全局变量
	SQLGroupManager = newGroupManager()
)

func newGroupManager() *GroupManager {
	return &GroupManager{
		groups: make(map[string]*Group),
	}
}

func (gm *GroupManager) Add(name string, g *Group) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.groups[name] = g
	return nil
}

func (gm *GroupManager) Get(name string) *Group {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.groups[name]
}

func (gm *GroupManager) PartitionBy(partiton func() (bool, string, string)) *Client {
	isMaster, dbName, tableName := partiton()
	return &Client{gm.Get(dbName).Instance(isMaster).Table(tableName)}
}

// Get(name)等于SQLGroupManager.Get(name)
func Get(name string) *Group {
	return SQLGroupManager.Get(name)
}

func PartitionBy(partiton func() (bool, string, string)) *Client {
	return SQLGroupManager.PartitionBy(partiton)
}
