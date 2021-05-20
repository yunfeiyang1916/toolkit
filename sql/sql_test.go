package sql

import (
	"os"
	"testing"

	log "github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/tomlconfig"
)

type groupConfig struct {
	Databases []struct {
		Name   string   `toml:"name"`
		Master string   `toml:"master"`
		Slaves []string `toml:"slaves"`
	} `toml:"database"`
}

type AllServerPolicy struct {
	ServerName      string `gorm:"server_name"`
	ServerType      int    `gorm:"server_type"`
	ServerURL       string `gorm:"server_url"`
	SecureServerURL string `gorm:"secure_server_url"`
}

// func (all AllServerPolicy) TableName() string {
// 	return "all_server_policy"
// }

var groupManager *GroupManager

func setUp() {
	config := "./sql_config.toml"
	var gc groupConfig
	err := tomlconfig.ParseTomlConfig(config, &gc)
	if err != nil {
		log.Errorf("toml config parse error %v", err)
		os.Exit(1)
	}
	// log.Debugf("groupConfig %+v", gc)
	groupManager = newGroupManager()
	for _, d := range gc.Databases {
		g, err := NewGroup(SQLGroupConfig{
			Name:   d.Name,
			Master: d.Master,
			Slaves: d.Slaves,
		})
		if err != nil {
			log.Errorf("init group error %v", err)
			os.Exit(1)
		}
		err = groupManager.Add(d.Name, g)
		if err != nil {
			log.Errorf("add group error %v", err)
			os.Exit(1)
		}
	}
	// log.Debugf("group manager %+v", groupManager)
}

func TestGroupCreate(t *testing.T) {
	setUp()
	if m := groupManager.Get("test1").Master(); m == nil {
		log.Fatal("get tet1 error")
	}
	groupManager.Get("test1").Slave()
	groupManager.Get("test2").Master()
	groupManager.Get("test2").Slave()
}

func TestQuery(t *testing.T) {
	setUp()
	var all []*AllServerPolicy
	err := groupManager.Get("test1").Master().Table("all_server_policy").Find(&all).Error
	if err != nil {
		log.Fatalf("query all error %v", err)
		os.Exit(1)
	}
	// log.Debugf("all response %+v error %v %+v", all, err, all[0])
}

func TestPartition(t *testing.T) {
	setUp()
	var all []*AllServerPolicy
	p := func() (bool, string, string) {
		return true, "test1", "all_server_policy"
	}
	err := groupManager.PartitionBy(p).Find(&all).Error
	if err != nil {
		log.Fatalf("query all error %v", err)
		os.Exit(1)
	}
	// log.Debugf("all response %+v error %v %+v", all, err, all[0])
}
