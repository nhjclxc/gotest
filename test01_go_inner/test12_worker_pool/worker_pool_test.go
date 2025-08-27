package test12_worker_pool

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"sync"
	"test12_worker_pool/workerpool"
	"testing"
	"time"
)

// 定义任务类型
type Task func()

// Worker Pool
type Pool struct {
	taskQueue chan Task
}

func NewPool(workerCount int) *Pool {
	p := &Pool{
		taskQueue: make(chan Task),
	}
	// 启动固定数量的 worker
	for i := 0; i < workerCount; i++ {
		go p.worker(i)
	}
	return p
}

func (p *Pool) worker(id int) {
	for task := range p.taskQueue {
		fmt.Printf("Worker %d start task\n", id)
		task()
		fmt.Printf("Worker %d finish task\n", id)
	}
}

// 提交任务
func (p *Pool) Submit(task Task) {
	p.taskQueue <- task
}

// 关闭池
func (p *Pool) Shutdown() {
	close(p.taskQueue)
}

func Test1(t *testing.T) {

	pool := NewPool(3) // 3个worker

	for i := 0; i < 10; i++ {
		n := i
		pool.Submit(func() {
			fmt.Printf("Task %d running\n", n)
			time.Sleep(time.Second)
		})
	}

	time.Sleep(5 * time.Second)
	pool.Shutdown()

}

func Test2(t *testing.T) {

	var wg sync.WaitGroup

	workPool := make(chan string, 5)

	// 启动消费者
	//go func() {
	//	for data := range workPool {
	//		fmt.Println("消费：" + data)
	//		wg.Done() // 每消费一个任务，Done 一次
	//	}
	//}()

	// 启动固定数量的 worker
	workerCount := 3
	for w := 0; w < workerCount; w++ {
		go func(id int) {
			for data := range workPool {
				fmt.Printf("Worker %d 消费：%s\n", id, data)
				wg.Done()
			}
		}(w)
	}

	// 提交任务
	for i := 0; i < 10; i++ {
		wg.Add(1)
		workPool <- "任务" + fmt.Sprintf("%d", i)
	}

	wg.Wait()
	close(workPool) // 关闭任务通道，让消费者退出
}

func Test3(t *testing.T) {

	// "github.com/panjf2000/ants/v2"
	p, _ := ants.NewPool(5) // 限制最多 5 个 goroutine
	defer p.Release()

	for i := 0; i < 10; i++ {
		n := i
		p.Submit(func() {
			fmt.Printf("Task %d running\n", n)
		})
	}
}

func Test5(t *testing.T) {
	// 创建一个 worker 数量=3，队列大小=10 的协程池
	pool := workerpool.NewPool(3, 10)

	// 提交任务
	for i := 0; i < 10; i++ {
		n := i
		pool.Submit(func() {
			fmt.Printf("执行任务 %d\n", n)
			time.Sleep(time.Second)
		})
	}

	// 不再接收新任务
	pool.Shutdown()

	// 等待任务全部完成
	pool.AwaitTermination()

	fmt.Println("所有任务执行完毕")
}
