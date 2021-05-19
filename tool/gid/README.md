# 分布式id生成器
### 特点
- 支持高并发
- 附带时间戳，相对有序
- 不依赖中心节点

### 使用
```go
package main

import (
	"git.inke.cn/BackendPlatform/gid"
	"fmt"
)

func main() {
	id := gid.New()
	fmt.Printf("16进制：%x\n", id)
	fmt.Printf("10进制：%d\n", id)

	s := fmt.Sprintf("%x", id)
	want := gid.UnixFromUint64(id)
	got := gid.UnixFromStr(s)
	if got != want {
		fmt.Printf("parse err,got %d,want %d", got, want)
	}
	fmt.Printf("解析时间戳：got %d,want %d\n", got, want)
}
/*
16进制：7a16ed3f66fb5
10进制：2147822211592117
解析时间戳：got 1554273261,want 1554273261
*/

```

### 生成方式

规则：28bit(duration)+16bit(iphash)+20bit(random num)

### 参考文档
[分布式id生成器](http://wiki.inkept.cn/pages/viewpage.action?pageId=57345037)