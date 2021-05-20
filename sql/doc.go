// sql为公司内部mysql client
//
// Background
//
// 这个库封装了http://gorm.io/
//
// Configure
//
// rpc-go框架提供初始化全局GroupManager, 配置项如下:
//	[[database]]
//		name="test1"
//		master = "admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"
//		slaves = ["admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"]
//	[[database]]
//		name="test2"
//		master = "admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"
//		slaves = ["admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"]
//
// 框架初始化完之后，可以使用Get函数来获取Group:
//  g := sql.Get("test1")
//  if g == nil {
//    log.Fatalf("group test1 is nil\n")
//  }
//
package sql
