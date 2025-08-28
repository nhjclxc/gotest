package main

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Task 任务结构体
type Task struct {
	Id       string
	Name     string
	Priority int       // 优先级，越大越优先
	Rate     int       // 进度 [0~10]
	AddTime  time.Time // 加入时间

	index int // heap内部用的索引
}

// String 格式化输出
func (t *Task) String() string {
	return fmt.Sprintf("任务[%s] 优先级:%d 进度:%d 加入时间:%s",
		t.Name, t.Priority, t.Rate, t.AddTime.Format("15:04:05"))
}

// -------------------- 优先级队列 --------------------

type TaskPriorityQueue []*Task

func (pq TaskPriorityQueue) Len() int { return len(pq) }

// 比较规则：优先级高 → 加入时间早
func (pq TaskPriorityQueue) Less(i, j int) bool {
	if pq[i].Priority == pq[j].Priority {
		return pq[i].AddTime.Before(pq[j].AddTime)
	}
	return pq[i].Priority > pq[j].Priority
}

func (pq TaskPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *TaskPriorityQueue) Push(x any) {
	n := len(*pq)
	task := x.(*Task)
	task.index = n
	*pq = append(*pq, task)
}

func (pq *TaskPriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	task := old[n-1]
	old[n-1] = nil // 避免内存泄漏
	task.index = -1
	*pq = old[0 : n-1]
	return task
}

// -------------------- 任务管理器 --------------------

type TaskManager struct {
	mu   sync.Mutex
	pq   TaskPriorityQueue
	cond *sync.Cond
}

func NewTaskManager() *TaskManager {
	tm := &TaskManager{}
	heap.Init(&tm.pq)
	tm.cond = sync.NewCond(&tm.mu)
	return tm
}

// 添加新任务
func (m *TaskManager) AddTask(name string, priority int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := &Task{
		Id:       uuid.New().String(),
		Name:     name,
		Priority: priority,
		Rate:     0,
		AddTime:  time.Now(),
	}
	heap.Push(&m.pq, t)
	fmt.Printf("✅ 新任务加入: %v\n", t)

	// 通知调度器有新任务
	m.cond.Signal()
}

// 调度器循环
func (m *TaskManager) Run() {
	for {
		m.mu.Lock()
		for m.pq.Len() == 0 {
			m.cond.Wait() // 没有任务时等待
		}
		// 取出优先级最高的任务
		task := heap.Pop(&m.pq).(*Task)
		m.mu.Unlock()

		// 模拟执行一个时间片
		m.doing(task)

		// 如果没完成，放回堆里
		if task.Rate < 10 {
			m.mu.Lock()
			heap.Push(&m.pq, task)
			m.mu.Unlock()
		} else {
			fmt.Printf("🎉 任务完成: %v\n", task)
		}
	}
}

// 执行一个时间片
func (m *TaskManager) doing(t *Task) {
	t.Rate++
	fmt.Printf("⚡ 正在处理: %v\n", t)
	time.Sleep(200 * time.Millisecond) // 一个时间片 200ms
}

// -------------------- main --------------------

func main() {
	m := NewTaskManager()
	go m.Run()

	// 模拟加入任务
	go func() {
		m.AddTask("任务1", 1)
		time.Sleep(500 * time.Millisecond)
		m.AddTask("任务2", 3)
		time.Sleep(500 * time.Millisecond)
		m.AddTask("任务3", 2)
		time.Sleep(500 * time.Millisecond)
		m.AddTask("任务4", 5)
	}()

	select {}
}

/*
好的 👍 我给你写一个 优先级堆 + 协作式抢占调度器 的完整示例。

思路是这样的：

用 container/heap 来维护一个 最大堆，堆里存放待运行的任务（排序规则：优先级高 > 加入时间早）。

调度器循环从堆中取出一个任务，执行一个时间片（slice）（比如加 1 个进度，耗时 100ms），然后：

如果任务没完成，就放回堆里。

如果完成了，就丢掉。

新任务通过 addNewTask 推入堆里，调度器会在下一次循环时自动考虑它。

这样高优先级任务随时能插队，因为堆会保证每次取出的都是当前优先级最高 + 最早加入的。
*/
