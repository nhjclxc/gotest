package main

import (
	"flag"
	"fmt"
	"gin_docker/config"
	"gin_docker/core"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

// 初识 gin 框架
func main() {

	var configFile string

	// 允许传入配置文件路径
	flag.StringVar(&configFile, "c", "", "choose config file")
	flag.Parse()

	if configFile == "" {
		slog.Error("未读取到配置文件，程序启动失败！！！", slog.String("configFile", configFile))
		os.Exit(-1)
	}
	slog.Info("成功读取到配置文件。", slog.String("configFile", configFile))

	config.InitConfig(configFile)

	fmt.Printf("成功读取到配置： %#v \n", config.GlobalConfig)

	// 初始化日志
	core.InitLogger()

	// 创建一个默认的路由引擎
	router := gin.Default()

	// 路由绑定
	router.GET("/news", func(context *gin.Context) {
		id := context.Query("id")
		name := context.Query("name")
		context.String(http.StatusOK, "我是 news 页面【【【2.0需求页面】】】，当前环境是：%#v，现在请求的文章id = %v, name = %v， \n", config.GlobalConfig, id, name)
	})

	// go run main.go
	// go run main.go -c config/config-dev.yaml
	// go run main.go -c config/config-test.yaml
	// go run main.go -c config/config-prod.yaml

	//启动端口监听
	//router.Run(":8090")
	router.Run(":" + strconv.Itoa(config.GlobalConfig.Port))
	//router.Run("localhost:8090")
}

/*
docker run -d --name gin_docker -p 19090:19090 -v $(pwd)/config.yaml:/app/config.yaml gin_docker
*/
