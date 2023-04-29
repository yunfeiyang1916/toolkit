package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/compiler/protogen"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

// protoc代码生成器插件
func main() {
	// 读取标准输入，接收proto解析的文件内容，并解析结构体
	input, _ := io.ReadAll(os.Stdin)
	var req pluginpb.CodeGeneratorRequest
	proto.Unmarshal(input, &req)
	// 生成插件
	opts := protogen.Options{}
	plugin, err := opts.New(&req)
	if err != nil {
		panic(err)
	}
	for _, file := range plugin.Files {
		// 创建一个buf 写入生成的文件内容
		var buf bytes.Buffer
		// 写入go文件的package名
		pkg := fmt.Sprintf("package %s", file.GoPackageName)
		buf.WriteString(pkg)

		// 遍历proto中的消息
		for _, msg := range file.Messages {
			// 为每个消息生成hello方法
			buf.WriteString(fmt.Sprintf(`
				func(c *%s)Hello(){

				}
			`, msg.GoIdent.GoName))
		}
		// 输出文件名
		filename := file.GeneratedFilenamePrefix + ".cloud.go"
		rFile := plugin.NewGeneratedFile(filename, ".")
		// 将内容写入插件文件内容
		rFile.Write(buf.Bytes())
	}
	// 生成响应
	stdout := plugin.Response()
	out, err := proto.Marshal(stdout)
	if err != nil {
		panic(err)
	}
	// 将响应写回标准输出，protoc会读取这个内容
	fmt.Fprintf(os.Stdout, string(out))
}
