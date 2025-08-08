package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	arpc "github.com/zyxar/argo/rpc"
)

const (
	aria2Url   = "http://localhost:6800/jsonrpc"
	aria2Token = "4hai1+"
	dst        = "/Users/lxc20250729/lxc/code/gotest/data"
	url        = "http://192.168.222.182:8484/BU-FENG-ZHUI-YING_FTR_S_CMN-QMS-EN_145M_51_2K_20250724_HXFILM_OV/BU-FENG-ZHUI-YING_FTR_S_CMN-QMS-EN_145M_51_2K_20250724_HXFILM_OV_audio_06.mxf"
	//url = "http://192.168.222.182:8484/BU-FENG-ZHUI-YING_FTR_S_CMN-QMS-EN_145M_51_2K_20250724_HXFILM_OV/BU-FENG-ZHUI-YING_FTR_S_CMN-QMS-EN_145M_51_2K_20250724_HXFILM_OV_07.mxf"
)

type Aria2Notifer struct {
	ch chan Aria2Event
}

type Aria2Event struct {
	Events    []arpc.Event
	EventName string
}

func (s *Aria2Notifer) OnDownloadStart(events []arpc.Event) {
	slog.Info("receive download start", slog.Any("events", events))
}

func (s *Aria2Notifer) OnDownloadPause(events []arpc.Event) {
	slog.Info("receive download pause", slog.Any("events", events))
}

func (s *Aria2Notifer) OnDownloadStop(events []arpc.Event) {
	// slog.Info("receive download stop........", slog.Any("events", events))

	// for _, event := range events {
	// 	slog.Info("download stop event details", slog.String("gid", event.Gid))
	// }

	// // 将停止事件也当作错误处理，让系统检查是否需要重试
	// s.ch <- Aria2Event{
	// 	EventName: "downloadError",
	// 	Events:    events,
	// }
}

func (s *Aria2Notifer) OnDownloadComplete(events []arpc.Event) {
	slog.Info("receive download complete", slog.Any("events", events))
	cp := make([]arpc.Event, len(events))
	copy(cp, events)
	s.ch <- Aria2Event{
		EventName: "downloadComplete",
		Events:    cp,
	}
}

func (s *Aria2Notifer) OnDownloadError(events []arpc.Event) {
	slog.Info("receive download error........", slog.Any("events", events))
	// s.ch <- Aria2Event{
	// 	EventName: "downloadError",
	// 	Events:    events,
	// }

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("OnDownloadError.recover.err", err)
		}
	}()

	if reCount > 11 {
		return
	}

	if arpcClient == nil {
		connect()
		fmt.Println("OnDownloadError.arpcClient.download, ", arpcClient)
	}

	for _, e := range events {

		st, err := arpcClient.TellStatus(e.Gid)
		if err != nil || len(st.Files) == 0 {
			continue
		}

		file := st.Files[0].Path

		fmt.Printf("download error : %s, failed reason: %s, file is = %s", e.Gid, st.ErrorMessage, file)

	}

	download(arpcClient, []string{url}, dst, map[string]interface{}{
		"dir": dst,
		//"always-resume": "true",
	})

	reCount++

}

var once sync.Once
var count int

func connect() {
	once.Do(func() {

		fmt.Println("connect.count = ", count)
		count++

		notifer := &Aria2Notifer{ch: make(chan Aria2Event, 1024)}
		c, err := arpc.New(context.Background(), aria2Url, aria2Token, time.Second*10, notifer)

		arpcClient = c

		if err != nil {
			fmt.Println("connect.arpc.New.err = ", err)
		}
		fmt.Println("连接成功：", c)

		params := map[string]interface{}{
			"http-user":                "feedying",
			"http-passwd":              "lambda3",
			"max-concurrent-downloads": 40,
			"dir":                      dst,
		}
		_, err = c.ChangeGlobalOption(params)
		if err != nil {
			slog.Error("new downloader change global options failed, retry after 10s",
				slog.Any("params", params), slog.Any("err", err))
		}

	})
}

func (s *Aria2Notifer) OnBtDownloadComplete(events []arpc.Event) {
	slog.Info("receive bt download complete", slog.Any("events", events))
}

var arpcClient arpc.Client
var reCount int

func main() {

	exitChan := make(chan bool)

	fmt.Println(url)

	connect()

	fmt.Println("arpcClient.download, ", arpcClient)
	go download(arpcClient, []string{url}, dst, map[string]interface{}{
		"dir": dst,
		//"always-resume": "true",
	})

	<-exitChan
}

func download(arpcClient arpc.Client, urls []string, dst string, options interface{}) {
	gid, err := arpcClient.AddURI(urls, options)
	if err != nil {
		slog.Error("aria2 AddURI failed",
			slog.Any("err", err),
			slog.String("dst", dst),
			slog.Any("urls", urls))
	}
	fmt.Println("gid = " + gid)
}

func continueDownload(arpcClient arpc.Client, events []arpc.Event) {

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("continueDownload.recover.err", err)
		}
	}()

	for _, event := range events {
		arpcClient.Unpause(event.Gid) // 继续下载（断点续传）
	}

}
