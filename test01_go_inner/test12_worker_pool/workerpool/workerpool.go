package workerpool

import (
	"sync"
)

// Pool 协程池
type Pool struct {
	taskQueue   chan func() // 任务队列
	wg          sync.WaitGroup
	onceClose   sync.Once
	workerCount int
}

// NewPool 创建一个固定大小的协程池
func NewPool(workerCount int, queueSize int) *Pool {
	p := &Pool{
		taskQueue:   make(chan func(), queueSize),
		workerCount: workerCount,
	}

	// 启动固定数量的 worker
	for i := 0; i < workerCount; i++ {
		go p.worker()
	}

	return p
}

// worker 消费任务
func (p *Pool) worker() {
	for task := range p.taskQueue {
		task()
		p.wg.Done()
	}
}

// Submit 提交任务
func (p *Pool) Submit(task func()) {
	p.wg.Add(1)
	p.taskQueue <- task
}

// Shutdown 关闭任务提交
func (p *Pool) Shutdown() {
	p.onceClose.Do(func() {
		close(p.taskQueue)
	})
}

// AwaitTermination 等待所有任务完成
func (p *Pool) AwaitTermination() {
	p.wg.Wait()
}
