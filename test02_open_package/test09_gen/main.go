package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type User struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Address string `json:"address"`
}

// 使用http.ListenAndServe()创建服务，使用gin.Default()绑定路由
func main() {

	//gin.SetMode(gin.ReleaseMode)
	//gin.SetMode(gin.TestMode)
	gin.SetMode(gin.DebugMode)

	// 创建一个默认的路由引擎
	router := gin.Default()

	// 路由绑定
	router.POST("/user", func(c *gin.Context) {
		var user1 User
		c.ShouldBindJSON(&user1)
		//c.ShouldBindBodyWithJSON(&user1)
		fmt.Printf("user1 %#v \n", user1)
		var user2 User
		c.ShouldBindJSON(&user2)
		//c.ShouldBindBodyWithJSON(&user2)
		fmt.Printf("user2 %#v \n", user2)
		var user3 User
		c.ShouldBindJSON(&user3)
		//c.ShouldBindBodyWithJSON(&user3)
		fmt.Printf("user3 %#v \n", user3)

		c.JSON(http.StatusOK, map[string]any{
			"code": 200,
			"msg":  "操作成功",
			"data": struct {
			}{},
		})
	})
	//
	////启动端口监听
	//router.Run(":8080")

	// 自定义 HTTP server 配置
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router, // 将 gin.Engine 绑定到 http.Server
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("服务启动: http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("服务器启动失败: %v\n", err)
	}

}
