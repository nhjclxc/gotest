package main

import (
	"context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
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
	r.GET("/live/flv/:brokerKey/:clientId", flvClient.LiveFlv(flvBroadcastPool))

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
	r.GET("/live/hls/:brokerKey/:clientId/*filepath", hlsClient.LiveHLS(hlsBroadcastPool))

	// ============== camera ==============,  先启动go服务器，再打开前端页面，最后使用ffmpeng推流
	cameraBroadcastPool = cameraBroadcast.NewCameraBroadcaster()
	brokerKey := "test-camera"
	var cameraM3U8Broker *cameraBroker.CameraBroker = cameraBroker.NewCameraBroker(brokerKey, 150)

	cameraBroadcastPool.AddBroker(brokerKey, cameraM3U8Broker)

	// ffmpeg -f avfoundation -framerate 30 -video_size 640x480 -i "0:0" -vcodec libx264 -preset veryfast -tune zerolatency -g 30 -acodec aac -ar 44100 -ac 2 -f flv "http://127.0.0.1:8080/live/camera/ingest/test-camera"
	// http://127.0.0.1:8080/live/camera/ingest/test-camera
	// camera ffmpeg 推流接口
	r.POST("/live/camera/ingest/:brokerKey", cameraClient.ExecutePush(cameraBroadcastPool))

	// http://127.0.0.1:8080/live/camera/test.flv
	// camera HTTP-FLV 拉流接口
	//r.GET("/live/:stream.flv", func(c *gin.Context) {
	r.GET("/live/camera/:brokerKey/:clientId", cameraClient.ExecutePull(cameraBroadcastPool))

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
