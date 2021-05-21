// 封装go语言redis client.
//
// Background
//
// 此基础库库基于redigo进行封装。之所以封装一个原生的库，是因为便于维护，容易扩展其他的东西：
//
// 1. 支持stst统计上报
//
// 2. 以基础库的形式，使大家的代码保持统一
//
// 3. 基础库统一维护
//
// 4. 支持trace
//
// Executing Commands
//
// Redis client有一个通用的方法来执行redis命令：
//
//  Do(commandName string, args ...interface{}) (reply interface{}, err error)
//
// Redis命令的参考(http://redis.io/commands) 列出了所有redis的命令，
// 一个使用Redis APPEND命令的例子：
//
//  n, err := r.Do("APPEND", "key", "value")
//
// Configure
//
// 此配置适用于rpc-go基础库中用文件进行初始化基础库:
//
//  [[redis]]
//	server_name="queen-redis"
//	addr="localhost:6379"
//	password="password"
//	max_idle=100
//	max_active=100
//	idle_timeout=1000
//	connect_timeout=1000
//	read_timeout=1000
//	write_timeout=1000
//	database=0
//	retry=0
//
// 更通用的配置方法请参考RedisConfig类型
//
// Trace
//
// 此redis client接入了公司内部的trace系统, 通过For函数进行接入:
//
//  ok, err := r.For(ctx).Set("key", "value")
//
// 更详细的内容请见For函数
package redis
