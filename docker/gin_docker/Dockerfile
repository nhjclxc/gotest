
# 1. 使用更小的镜像进行部署，Alpine是一个轻量级的Linux发行版，超小型 Linux 发行版
# FROM alpine:latest
# https://docker.aityp.com/image/docker.io/library/alpine:latest
# docker pull swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:latest
# docker tag  swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:latest  docker.io/library/alpine:latest
FROM docker.io/library/alpine:latest

# 进入docker内部的工作目录
WORKDIR /app

# 设置时区
RUN uname -m
RUN rm -f /etc/localtime && ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

# 将编译好的可执行文件拷贝到镜像里面
COPY ./gin_docker /app/gin_docker

# 给应用添加所有权限
RUN chmod 777 /app/gin_docker

# 暴露端口（根据项目端口修改）EXPOSE 本身不会真正把端口暴露到宿主机，它只是用来声明容器内部会监听哪些端口。
EXPOSE 18080 18090 19090

# 设定启动命令
CMD ["/app/gin_docker", "-c", "/app/config.yaml"]
