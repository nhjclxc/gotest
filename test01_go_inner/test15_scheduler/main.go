package main

import (
	"cmp"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"slices"
	"sync"
	"time"
)

// 实现下载调度器, 要求：优先级越大越优先被执行；如果优先级相同，则按加入时间先后来下载。
// 要求：优先级越大越优先被执行；如果优先级相同，则按加入时间先后来下载。

// Task 任务结构体
type Task struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	Priority int       `json:"priority"`
	Rate     int       `json:"rate"`
	AddTime  time.Time `json:"addTime"`
}

func (t Task) String() string {
	return fmt.Sprintf("任务：%s，优先级：%d，进度：%d，加入时间：%s \n", t.Name, t.Priority, t.Rate, t.AddTime.Format("2006-01-02 15:04:05"))
}

// TaskManager 任务管理器
type TaskManager struct {
	Lock          sync.Mutex
	TodoTaskPool  map[string]*Task // 待执行的任务
	RunningTaskId string           // 正在执行的任务
	newTaskCh     chan *Task
}

// GetHighestPriorityTask 寻找任务池中优先级和最旧的一个任务来执行
func (m *TaskManager) GetHighestPriorityTask() *Task {
	if len(m.TodoTaskPool) <= 0 {
		return nil
	}

	m.Lock.Lock()
	defer m.Lock.Unlock()

	// 1️⃣ 取 map 的值到切片
	tasks := make([]*Task, 0, len(m.TodoTaskPool))
	for _, task := range m.TodoTaskPool {
		tasks = append(tasks, task)
	}

	slices.SortFunc(tasks, func(a, b *Task) int {
		if r := cmp.Compare(b.Priority, a.Priority); r != 0 {
			return r
		}
		return cmp.Compare(a.AddTime.UnixNano(), b.AddTime.UnixNano())
	})

	return tasks[0]
}

func (m *TaskManager) work() {
	fmt.Println("work启动成功，等待任务到来！！！")
	for {
		select {
		case newT, ok := <-m.newTaskCh:
			if ok {
				m.RunningTaskId = newT.Id
				m.doing(newT)
			}
		default:
			if m.RunningTaskId != "" {
				m.doing(m.TodoTaskPool[m.RunningTaskId])
			} else {
				// 没有新任务到来，并且当前也没有正在执行的任务了，那么将任务池里面的任务拿出来执行
				task := m.GetHighestPriorityTask()
				if task != nil {
					m.RunningTaskId = task.Id
					m.doing(task)
				}
			}
		}
	}
}

func (m *TaskManager) doing(t *Task) {
	t.Rate += 1
	fmt.Printf("当前处理的任务是：%v \n", t)
	time.Sleep(1000 * time.Millisecond)
	if t.Rate >= 10 {
		m.Lock.Lock()
		m.RunningTaskId = ""
		delete(m.TodoTaskPool, t.Id)
		m.Lock.Unlock()
		fmt.Printf("任务：%v 处理完毕！！！\n", t)
	}
}

// addNewTask 新增一个任务
func (m *TaskManager) addNewTask(newT *Task) {

	// 将当前任务加入任务池
	m.Lock.Lock()
	m.TodoTaskPool[newT.Id] = newT
	m.Lock.Unlock()

	// 检查现在是否有任务
	if m.RunningTaskId == "" {
		// 当前没有任务在运行，直接开始当前任务
		m.newTaskCh <- newT
		return
	}

	// 判断当前下载任务的优先级和当前新增的任务的优先级哪个高
	poolTask := m.GetHighestPriorityTask()
	runningTask := m.TodoTaskPool[m.RunningTaskId]
	if poolTask.Priority > runningTask.Priority {
		m.newTaskCh <- poolTask
	}

}

func main() {

	m := TaskManager{
		TodoTaskPool: make(map[string]*Task, 0),
		newTaskCh:    make(chan *Task),
	}
	go m.work() // 开启协程等待任务的到来

	r := gin.Default()

	r.POST("add", func(c *gin.Context) {
		var t Task
		c.ShouldBindJSON(&t)

		fmt.Printf("新任务：%#v \n", t)
		t.AddTime = time.Now()
		t.Id = uuid.New().String()

		go m.addNewTask(&t)

		fmt.Printf("新任务 %#v 添加成功 ！！！\n", t)
	})

	/*
		http://localhost:8080/add
		{
			"name": "任务5",
			"priority": 5
		}
	*/

	r.Run(":8080")

}

// 1, 11, 21, 15, 5
