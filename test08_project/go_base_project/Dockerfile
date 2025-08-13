# FROM alpine:latest
# https://docker.aityp.com/image/docker.io/library/alpine:latest
# docker pull swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:latest
# docker tag  swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:latest  docker.io/library/alpine:latest
FROM docker.io/library/alpine:latest
WORKDIR /app


RUN uname -m
RUN rm -f /etc/localtime && ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY ./bin/go_base_project /app/go_base_project
RUN chmod 777 /app/go_base_project

CMD ["/app/go_base_project", "serve", "-c", "/app/config.yaml"]
