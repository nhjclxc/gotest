#!/bin/bash

# 设置目标系统和架构，并构建 Go 项目为 Linux 下的 gin_docker 可执行文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gin_docker main.go

# 检查构建是否成功
if [ $? -eq 0 ]; then
    echo "✅ Build successful: gin_docker"
else
    echo "❌ Build failed"
    exit 1
fi
