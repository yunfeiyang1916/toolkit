#!/bin/bash
# 生成go model文件和go grpc服务桩代码
protoc --proto_path=../user/ user.proto --go_out=./server --go-grpc_out=./server
