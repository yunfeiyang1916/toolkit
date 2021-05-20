package sql_test

import (
	"os"

	log "github.com/yunfeiyang1916/toolkit/logging"
	"github.com/yunfeiyang1916/toolkit/sql"
	"github.com/yunfeiyang1916/toolkit/tomlconfig"
)

func ExampleClient() {
	g := sql.Get("group1")
	if g == nil {
		log.Fatalf("group is nil")
	}

	type TabConfigSql struct {
		Priority     float64 `gorm:"priority"`
		White_users  string  `gorm:"white_users"`
		Android_vers string  `gorm:"android_vers"`
		Ios_vers     string  `gorm:"ios_vers"`
		Locations    string  `gorm:"locations"`
		Tails        string  `gorm:"tails"`
		Between      string  `gorm:"between"`
		Tabs         string  `gorm:"tabs"`
		SubTabs      string  `gorm:"sub_tabs"`
	}

	var all []*TabConfigSql

	err := g.Instance(true).
		Table("tab_config").
		Where("id = 1").
		Order("priority desc").
		Find(&all).Error

	if err != nil {
		log.Fatalf("err=%+v\n", err)
	}
}

func ExampleNewGroup() {
	g, err := sql.NewGroup(sql.SQLGroupConfig{
		Name:   "db1",                                                         // database名称
		Master: "user:password@/dbname?charset=utf8&parseTime=True&loc=Local", // master
		Slaves: []string{
			"user:password@/dbname?charset=utf8&parseTime=True&loc=Local", // slaves
		},
	})

	if err != nil {
		log.Fatalf("new group error %v", err)
	}

	type AllServerPolicy struct {
		ServerName      string `gorm:"server_name"`
		ServerType      int    `gorm:"server_type"`
		ServerURL       string `gorm:"server_url"`
		SecureServerURL string `gorm:"secure_server_url"`
	}

	var all []*AllServerPolicy

	err = g.Master().Table("all_server_policy").Find(&all).Error
	if err != nil {
		log.Fatalf("query all error %v", err)
	}
}

func ExampleGroupManager() {
	type groupConfig struct {
		Databases []sql.SQLGroupConfig `toml:"database"`
	}
	var gc groupConfig

	// sql_config.toml
	//	[[database]]
	//		name="test1"
	//		master = "admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"
	//		slaves = ["admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"]
	//
	//	[[database]]
	//		name="test2"
	//		master = "admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"
	//		slaves = ["admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"]

	err := tomlconfig.ParseTomlConfig("sql_config.toml", &gc)
	if err != nil {
		log.Fatalf("parse toml err %+v\n", err)
	}

	for _, d := range gc.Databases {
		g, err := sql.NewGroup(sql.SQLGroupConfig{
			Name:   d.Name,
			Master: d.Master,
			Slaves: d.Slaves,
		})
		if err != nil {
			log.Errorf("init group error %v", err)
			os.Exit(1)
		}
		err = sql.SQLGroupManager.Add(d.Name, g)
		if err != nil {
			log.Errorf("add group error %v", err)
			os.Exit(1)
		}
	}
}

func ExampleSQLGroupManager() {
	g, err := sql.NewGroup(
		sql.SQLGroupConfig{
			Name:   "db1",                                                         // database名称
			Master: "user:password@/dbname?charset=utf8&parseTime=True&loc=Local", // master
			Slaves: []string{
				"user:password@/dbname?charset=utf8&parseTime=True&loc=Local", // slaves
			},
		})

	if err != nil {
		log.Fatalf("new group error %v", err)
	}

	sql.SQLGroupManager.Add("db1", g)

	group := sql.SQLGroupManager.Get("db1")
	if group == nil {
		log.Fatalf("group is nil error")
	}
	//use group
}
