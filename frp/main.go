package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// 初识 gin 框架
func main() {

	// 创建一个默认的路由引擎
	router := gin.Default()

	// 路由绑定
	router.GET("/news", func(context *gin.Context) {
		id := context.Query("id")
		name := context.Query("name")
		context.String(http.StatusOK, "我是 news 页面，现在请求的文章id = %v, name = %v， \n", id, name)
	})

	//启动端口监听
	router.Run(":8090")
}
