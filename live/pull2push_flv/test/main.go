package test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// golang实现拉流转推

const (
	hlsDir        = "/tmp/hls"      // HLS 文件输出目录（可改）
	playlistFile  = "playlist.m3u8" // m3u8 文件名
	segmentPrefix = "segment_"      // ts 文件前缀
	// 调整参数：
	segmentDurationSec = 1
	playlistSize       = 3 // 缓存约 3s
)

// startFFmpeg 启动 ffmpeg 拉流并输出 HLS 到 hlsDir，返回 Cmd（未 Wait）
// src 可为 rtmp://... / rtsp://... / http://...
func startFFmpeg(ctx context.Context, src string) (*exec.Cmd, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(hlsDir, 0755); err != nil {
		return nil, err
	}

	segPattern := filepath.Join(hlsDir, segmentPrefix+"%05d.ts")
	outPlaylist := filepath.Join(hlsDir, playlistFile)

	// ffmpeg 参数：-c copy 保持原码流以降低 CPU
	args := []string{
		"-i", src,
		"-c", "copy",
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segmentDurationSec),
		"-hls_list_size", fmt.Sprintf("%d", playlistSize),
		"-hls_flags", "delete_segments+append_list",
		"-hls_segment_filename", segPattern,
		outPlaylist,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// 将 ffmpeg 的 stderr 打印出来，方便调试
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 打印 ffmpeg 输出（异步）
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := stderr.Read(b)
			if n > 0 {
				log.Printf("[ffmpeg][stderr] %s", string(b[:n]))
			}
			if err != nil {
				if err != io.EOF {
					// ignore
				}
				break
			}
		}
	}()
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := stdout.Read(b)
			if n > 0 {
				log.Printf("[ffmpeg][stdout] %s", string(b[:n]))
			}
			if err != nil {
				if err != io.EOF {
				}
				break
			}
		}
	}()

	return cmd, nil
}

// monitorAndRestartFFmpeg: 在后台运行 ffmpeg，并在退出时重启
func monitorAndRestartFFmpeg(ctx context.Context, src string, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			cmdCtx, cancel := context.WithCancel(ctx)
			cmd, err := startFFmpeg(cmdCtx, src)
			if err != nil {
				log.Printf("start ffmpeg error: %v; retry in 3s", err)
				cancel()
				time.Sleep(3 * time.Second)
				continue
			}
			log.Printf("ffmpeg started (pid=%d)", cmd.Process.Pid)

			// 等待 ffmpeg 结束
			err = cmd.Wait()
			cancel()
			if err != nil {
				log.Printf("ffmpeg exited with error: %v; restarting in 2s", err)
			} else {
				log.Printf("ffmpeg exited normally; restarting in 2s")
			}
			time.Sleep(2 * time.Second)
		}
	}()
}

func main01() {
	// 源地址，示例（请替换为你的直播源）
	// 支持 rtmp/rtsp/http(s) 等
	src := "http://192.168.203.182:8080/live/livestream.m3u8"

	// 如果想从环境变量传入
	if v := os.Getenv("PULL_SRC"); v != "" {
		src = v
	}

	// 启动 ffmpeg 管理协程
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	monitorAndRestartFFmpeg(ctx, src, &wg)

	// Gin 服务：暴露 HLS playlist & ts 文件
	router := gin.Default()

	// 1) 简单的静态文件服务（直接把 /tmp/hls 作为静态目录）
	router.Static("/live", hlsDir)

	// 2) 可选：更严格的缓存控制与允许查看目录
	router.GET("/", func(c *gin.Context) {
		c.String(200, "Go HLS proxy running. playlist at /live/"+playlistFile)
	})
	// http://localhost:8080/live/playlist.m3u8

	// 3) 如果希望内存缓存 ts，提高并发读取速度（可选）
	//    这里给出一个简单的示例：将最近 N 个 ts 读入内存缓冲（非必须）
	//    为了示例简短，此处不实现自动加载逻辑；默认使用静态文件即可。

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Printf("HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待退出信号（可自行扩展优雅关闭）
	stop := make(chan os.Signal, 1)
	// signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// 关闭流程
	cancel()
	shutdownCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_ = srv.Shutdown(shutdownCtx)
	wg.Wait()
}
