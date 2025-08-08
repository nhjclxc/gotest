# 1. 使用官方 Go 镜像作为构建阶段
# FROM golang:1.22 AS builder
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.22.3 AS builder

# 设置 Go 代理
ENV GOPROXY=https://goproxy.cn,direct

# 2. 设置工作目录
WORKDIR /app

# 3. 将 go.mod 和 go.sum 拷贝进去，并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 4. 拷贝源代码
COPY . .

# 5. 编译可执行文件（关闭 CGO 更适用于部署）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server main.go

# 6. 使用更小的镜像进行部署
# FROM alpine:latest
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:latest

WORKDIR /root/

# 拷贝编译好的文件到最终镜像  # 配置文件的文件夹开关
COPY --from=builder /app/server .
COPY --from=builder /app/config ./config

# 暴露端口（根据项目端口修改）EXPOSE 本身不会真正把端口暴露到宿主机，它只是用来声明容器内部会监听哪些端口。
EXPOSE 18080 18090 19090


# 设定启动命令
CMD ["./server"]
