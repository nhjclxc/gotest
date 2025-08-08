package core

import (
	"github.com/gin-gonic/gin"
	"io"
	"os"
)

func InitLogger() {
	// 创建目录（如果不存在）
	os.MkdirAll("logs", 0755)

	// 打开日志文件
	logFile, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("无法打开日志文件: " + err.Error())
	}

	// 同时写入文件和控制台
	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)
}
