package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ---------------- Worker 示例 ----------------
func worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	fmt.Printf("Worker %d 启动\n", id)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d 接收到退出信号\n", id)
			return
		default:
			// 模拟工作
			fmt.Printf("Worker %d 正在工作...\n", id)
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// ---------------- 通用运行器 ----------------
func runWorkers(ctx context.Context, n int) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go worker(ctx, wg, i)
	}
	return wg
}

// ---------------- 主程序 ----------------
func main() {
	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())

	// 捕捉系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动 worker
	wg := runWorkers(ctx, 3)

	// 等待退出信号
	go func() {
		sig := <-sigCh
		fmt.Printf("收到退出信号: %v\n", sig)
		cancel() // 通知所有 worker 退出
	}()

	// 阻塞等待所有 worker 完成
	wg.Wait()
	fmt.Println("所有 worker 已优雅退出，程序结束")
}
