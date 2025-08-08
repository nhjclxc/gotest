# 统一设置
APP_NAME := gin_docker
#VERSION  := $(shell date +%Y%m%d%H)
VERSION  ?= $(shell date +%Y%m%d%H)
# ?=表示接受一个参数

# 构建镜像
build:
	docker build -t $(APP_NAME):$(VERSION) .
	#make clean-images

# 启动 dev 环境容器
run-dev: build
	docker stop gin_docker-dev || true
	docker rm gin_docker-dev || true
	docker run -d \
		--name gin_docker-dev \
		-p 18080:18080 \
		-v $(PWD)/config:/app/config \
		-v $(PWD)/logs:/app/logs \
		$(APP_NAME):$(VERSION) ./server -c config/config-dev.yaml


# 启动 test 环境容器
run-test: build
	docker stop gin_docker-test || true
	docker rm gin_docker-test || true
	docker run -d \
		--name gin_docker-test \
		-p 18090:18090 \
		-v $(PWD)/config:/app/config \
		-v $(PWD)/logs:/app/logs \
		$(APP_NAME):$(VERSION) ./server -c config/config-test.yaml

# 启动 prod 环境容器
run-prod: build
	docker stop gin_docker-prod || true
	docker rm gin_docker-prod || true
	docker run -d \
		--name gin_docker-prod \
		-p 19090:19090 \
		-v $(PWD)/config:/app/config \
		-v $(PWD)/logs:/app/logs \
		$(APP_NAME):$(VERSION) ./server -c config/config-prod.yaml

# 清理所有容器
clean:
	docker rm -f gin_docker-dev gin_docker-test gin_docker-prod || true

clean-images:
	@echo "🧹 清理旧镜像（保留最新3个）..."
	@docker images | grep $(APP_NAME) | awk 'NR>3 {print $$3}' | xargs -r docker rmi -f


#make run-dev VERSION=12345     # 一键构建 + 启动 dev 环境
#make run-test VERSION=12345    # 一键构建 + 启动 test 环境
#make run-prod VERSION=12345    # 一键构建 + 启动 prod 环境
#make clean       # 删除所有容器
#make clean-images  # 删除旧镜像

