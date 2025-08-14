package middleware

import (
	"github.com/gin-gonic/gin"
	"log"
	"runtime/debug"
)

func GlobalPanicRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 这里写自定义日志处理
				log.Printf("panic recovered: %v\n", err)

				//		// 打印堆栈
				log.Printf("stack trace:\n%s", debug.Stack())
				// 发送告警、写日志等

				// 返回自定义错误响应
				c.AbortWithStatusJSON(500, gin.H{
					"message": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}
