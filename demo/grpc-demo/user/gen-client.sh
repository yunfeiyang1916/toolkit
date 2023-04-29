#!/bin/bash
# 生成go model文件（生成存根文件）
protoc --proto_path=../user/ user.proto --go_out=./client --go-grpc_out=./client
