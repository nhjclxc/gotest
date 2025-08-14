package main

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	cameraBroadcast "pull2push/core/broadcast/camera"
	flvBroadcast "pull2push/core/broadcast/flv"
	hlsBroadcast "pull2push/core/broadcast/hls"
	cameraBroker "pull2push/core/broker/camera"
	flvBroker "pull2push/core/broker/flv"
	hlsBroker "pull2push/core/broker/hls"
	cameraClient "pull2push/core/client/camera"
	flvClient "pull2push/core/client/flv"
	hlsClient "pull2push/core/client/hls"
	"pull2push/middleware"
	"strings"
	"time"
)

/*
使用ffmpeg推流：ffmpeg -re -i demo.flv -c copy -f flv rtmp://192.168.203.182/live/livestream

拉流
● RTMP (by VLC): rtmp://192.168.203.182/live/livestream
● H5(HTTP-FLV): http://192.168.203.182:8080/live/livestream.flv
● H5(HLS): http://192.168.203.182:8080/live/livestream.m3u8

	ffmpeg -re -i demo.flv \
	    -c:v libx264 -preset veryfast -tune zerolatency \
	    -g 25 -keyint_min 25 \
	    -c:a aac -ar 44100 -b:a 128k \
	    -f flv rtmp://192.168.203.182/live/livestream
*/

var (
	flvBroadcastPool    *flvBroadcast.FLVBroadcaster
	hlsBroadcastPool    *hlsBroadcast.HLSBroadcaster
	cameraBroadcastPool *cameraBroadcast.CameraBroadcaster
)

func main() {

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.GlobalPanicRecoveryMiddleware())

	// 添加 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "android-device-id", "Content-Type", "Accept", "Authorization", "X-Token", "request-time", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// health
	r.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ============== flv ==============
	flvBrokerKey := "test1"
	flvUpstreamURL := "http://192.168.203.182:8080/live/livestream.flv"

	flvBroadcastPool = flvBroadcast.NewFLVBroadcaster()
	var flvStreamBroker *flvBroker.FLVStreamBroker = flvBroker.NewFLVStreamBroker(flvBrokerKey, flvUpstreamURL)
	flvBroadcastPool.AddBroker(flvBrokerKey, flvStreamBroker)

	// http://localhost:8080/live/flv
	r.GET("/live/flv/:brokerKey/:clientId", LiveFlv())

	// ============== hls ==============

	hlsBrokerKey := "test-hls"
	hlsUpstreamURL := "http://192.168.203.182:8080/live/livestream.m3u8"

	hlsBroadcastPool = hlsBroadcast.NewBroadcaster()
	var hlsM3U8Broker *hlsBroker.HLSM3U8Broker = hlsBroker.NewHLSM3U8Broker(ctx, hlsBrokerKey, hlsUpstreamURL, "", 3)
	hlsBroadcastPool.AddBroker(hlsBrokerKey, hlsM3U8Broker)

	// hls要提供两个接口，一个是 index.m3u8用于客户端第一次调用的时候获取最新数据分片消息的，有助于第二个接口来获取最新的分片数据
	// 一个是 类似 2689.ts 的接口，用于给客户端请求具体的流数据
	// http://localhost:8080/live/hls/:brokerKey/:clientId/index.m3u8
	// http://localhost:8080/live/hls/:brokerKey/:clientId/2689.ts
	r.GET("/live/hls/:brokerKey/:clientId/*filepath", LiveHLS())

	// ============== camera ==============
	// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/camera/ingest/test-camera"
	// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/camera/ingest/test-camera"

	// http://127.0.0.1:8080/live/camera/ingest/test-camera
	// ffmpeg 推流接口
	r.POST("/live/camera/ingest/:stream", func(c *gin.Context) {
		//if !strings.HasPrefix(c.GetHeader("Content-Type"), "video/x-flv") {
		//	c.String(http.StatusBadRequest, "Content-Type must be video/x-flv")
		//	return
		//}

		cameraBroadcastPool = cameraBroadcast.NewCameraBroadcaster()
		var cameraM3U8Broker *cameraBroker.CameraBroker = cameraBroker.NewCameraBroker(c, 150)

		cameraBroadcastPool.AddBroker(hlsBrokerKey, cameraM3U8Broker)

	})

	// http://127.0.0.1:8080/live/camera/test.flv
	// HTTP-FLV 拉流接口
	//r.GET("/live/:stream.flv", func(c *gin.Context) {
	r.GET("/live/camera/:brokerKey/:clientId", func(c *gin.Context) {
		brokerKey := c.Param("brokerKey")
		clientId := c.Param("clientId")
		broker, err := cameraBroadcastPool.FindBroker(brokerKey)
		if err != nil {
			fmt.Printf("未找到对应的广播器 %s \n", brokerKey)
			return
		}

		cameraBroker, _ := broker.(*cameraBroker.CameraBroker)
		client, err := cameraClient.NewCameraLiveClient(c, brokerKey, clientId, cameraBroker.ClientCloseSig, cameraBroker.BrokerCloseSig)
		if err != nil {
			return
		}
		cameraBroker.AddLiveClient(clientId, client)

	})

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// LiveHLS 处理 hls 的拉流转推
func LiveHLS() func(c *gin.Context) {
	return func(c *gin.Context) {

		brokerKey := c.Param("brokerKey")
		clientId := c.Param("clientId")

		broker, err := hlsBroadcastPool.FindBroker(brokerKey)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code": 404,
				"msg":  "直播不存在！！！",
			})
			return
		}

		hlsM3U8Broker, _ := broker.(*hlsBroker.HLSM3U8Broker)

		filepath := c.Param("filepath")
		//  "xxx/index.m3u8" 结尾的就是第一次请求，这时通过 HandleIndex 接口第一次返回本地缓存的数据片给前端使用
		if strings.HasSuffix(filepath, "/index.m3u8") {

			hlsLiveClient, err := hlsClient.NewHLSLiveClient(c, brokerKey, clientId, hlsM3U8Broker.ClientCloseSig, hlsM3U8Broker.BrokerCloseSig)
			if err != nil {
				c.JSON(500, err)
				return
			}
			hlsM3U8Broker.AddLiveClient(clientId, hlsLiveClient)

			// 第一次链接，返回最新的直播数据分片
			hlsLiveClient.HandleIndex(c.Writer, c.Request, hlsM3U8Broker)

			return
		}
		// xxx/2689.ts 表示此时前端不是第一次掉接口了，是来获取缓存数据分片的，那么通过 HandleSegment 来下载数据分片

		liveClient, err := hlsM3U8Broker.FindLiveClient(clientId)
		if err != nil {
			c.JSON(500, err)
			return
		}
		hlsLiveClient, _ := liveClient.(*hlsClient.HLSLiveClient)
		// 返回本地缓存的数据分片
		hlsLiveClient.HandleSegment(c.Writer, c.Request, hlsM3U8Broker)
		// /live/hls/test-hls/c91b431e-ba21-47c9-8649-a05ce2490838/index.m3u8

	}
}

// LiveFlv 处理 flv 的拉流转推
func LiveFlv() func(c *gin.Context) {
	return func(c *gin.Context) {

		c.Header("Content-Type", "video/x-flv")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Transfer-Encoding", "chunked")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Cache-Control", "no-cache")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		// 确保响应缓冲区被刷新
		c.Writer.Flush()

		brokerKey := c.Param("brokerKey")
		clientId := c.Param("clientId")

		broker, _ := flvBroadcastPool.FindBroker(brokerKey)
		flvStreamBroker, _ := broker.(*flvBroker.FLVStreamBroker)
		flvStreamBroker.RemoveLiveClient(clientId)

		// 阻塞客户端
		//<-c.Request.Context().Done()

		//// 或者使用以下逻辑
		c.Stream(func(w io.Writer) bool {

			liveFLVClient, err := flvClient.NewFLVLiveClient(c, brokerKey, clientId, flvStreamBroker.DataCh, flvStreamBroker.GOPCache, flvStreamBroker)
			if err != nil {
				c.JSON(500, err)
				return false
			}

			flvStreamBroker.AddLiveClient(clientId, liveFLVClient)

			// 这里写数据推送逻辑，或者直接阻塞直到连接关闭
			<-c.Request.Context().Done()
			return false
		})

	}
}
