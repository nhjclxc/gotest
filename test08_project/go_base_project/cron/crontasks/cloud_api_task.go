package crontasks

import (
	"context"
	"encoding/json"
	"go_base_project/event"
	"go_base_project/logger"
	"go_base_project/resource"
	"net/http"
	"sync"
	"time"
)

// CloudAPITask 从云端API获取拉流转推的任务
type CloudAPITask struct {
	client       *http.Client
	resource     *resource.Resource
	eventBus     *event.EventBus
	interval     time.Duration
	requestURL   string
	currentTasks map[int]event.StreamCreateRequestPayload // 当前任务缓存，以LiveID为key
	mu           sync.RWMutex                             // 保护currentTasks
}

// NewCloudAPITask 创建一个新的云端API请求任务
func NewCloudAPITask(res *resource.Resource, bus *event.EventBus) *CloudAPITask {
	// 从配置中读取设置
	cfg := res.Config.Cronjob.CloudAPI
	timeout := time.Duration(cfg.Timeout) * time.Second
	interval := time.Duration(cfg.Interval) * time.Second

	return &CloudAPITask{
		client: &http.Client{
			Timeout: timeout,
		},
		resource:     res,
		eventBus:     bus,
		requestURL:   cfg.RequestURL,
		interval:     interval,
		currentTasks: make(map[int]event.StreamCreateRequestPayload),
	}
}

// Name 返回任务名称
func (t *CloudAPITask) Name() string {
	return "cloud_api_task"
}

func (t *CloudAPITask) Enable() bool {
	return t.resource.Config.Cronjob.CloudAPI.Enable
}

// Execute 执行任务
func (t *CloudAPITask) Execute(ctx context.Context) error {
	if !t.resource.Config.Cronjob.CloudAPI.Enable {
		logger.Info("CloudAPITask is disabled")
		return nil
	}

	logger.Info("Requesting live forward from cloud API")

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", t.requestURL, nil)
	if err != nil {
		logger.Error("Failed to create request", "error", err)
		return err
	}

	req.Header.Set("proprietarycoding", t.resource.Proprietarycoding)
	// 发送请求
	/*
			{
		    "data": {
		        "lives": [
		            {
		                "live_id": 1,
		                "source_url": "http-flv",
		                "live_name": "测试直播"
		            }
		        ],
		        "count": 1
		    },
		    "code": 0
		}
	*/
	resp, err := t.client.Do(req)
	if err != nil {
		logger.Error("Failed to send request", "error", err)
		return err
	}
	defer resp.Body.Close()

	type CloudAPIResponse struct {
		Data struct {
			Lives             []event.StreamCreateRequestPayload `json:"lives"`
			Proprietarycoding string                             `json:"proprietarycoding"`
		} `json:"data"`
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	// 解析响应
	var apiResp CloudAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		logger.Error("Failed to decode response", "error", err)
		return err
	}

	// 以debug级别打印接口返回值
	logger.Debug("Cloud API response", "response", apiResp)

	// 检查响应状态码
	if apiResp.Code != 0 {
		logger.Error("Cloud API returned error", "code", apiResp.Code, "msg", apiResp.Msg, "response", apiResp)
		return nil
	}

	// 验证任务数据有效性
	var validTasks []event.StreamCreateRequestPayload
	for _, task := range apiResp.Data.Lives {
		if task.LiveID == 0 || task.SourceURL == "" {
			logger.Warn("Invalid task data, skipping", "liveID", task.LiveID, "sourceURL", task.SourceURL, "task", task)
			continue
		}
		validTasks = append(validTasks, task)
	}

	// 对比任务并处理变更
	t.processTaskChanges(validTasks)

	return nil
}

func (t *CloudAPITask) GetInterval() time.Duration {
	return t.interval
}

// processTaskChanges 处理任务变更：新增、删除、更新
func (t *CloudAPITask) processTaskChanges(newTasks []event.StreamCreateRequestPayload) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 将新任务转换为map，便于查找
	newTasksMap := make(map[int]event.StreamCreateRequestPayload)
	for _, task := range newTasks {
		newTasksMap[task.LiveID] = task
	}

	// 1. 检查需要删除的任务（当前有但新任务中没有的）
	var tasksToDelete []int
	for liveID := range t.currentTasks {
		if _, exists := newTasksMap[liveID]; !exists {
			tasksToDelete = append(tasksToDelete, liveID)
			logger.Info("Task marked for deletion", "liveID", liveID)
		}
	}

	// 2. 检查需要新增或更新的任务
	var tasksToCreate []event.StreamCreateRequestPayload
	var tasksToUpdate []event.StreamCreateRequestPayload

	for liveID, newTask := range newTasksMap {
		if currentTask, exists := t.currentTasks[liveID]; exists {
			// 任务已存在，检查是否需要更新
			if currentTask.SourceURL != newTask.SourceURL {
				tasksToUpdate = append(tasksToUpdate, newTask)
				logger.Info("Task marked for update", "liveID", liveID, "oldURL", currentTask.SourceURL, "newURL", newTask.SourceURL)
			}
		} else {
			// 新任务
			tasksToCreate = append(tasksToCreate, newTask)
			logger.Info("Task marked for creation", "liveID", liveID, "sourceURL", newTask.SourceURL)
		}
	}

	// 3. 执行删除操作
	for _, liveID := range tasksToDelete {
		t.eventBus.Publish(event.Event{
			Type:    event.StreamStopRequest,
			Payload: liveID,
		})
		delete(t.currentTasks, liveID)
	}

	// 4. 执行更新操作（先停止再创建）
	for _, task := range tasksToUpdate {
		// 先停止旧流
		t.eventBus.Publish(event.Event{
			Type:    event.StreamStopRequest,
			Payload: task.LiveID,
		})
		// 再创建新流
		t.eventBus.Publish(event.Event{
			Type:    event.StreamCreateRequest,
			Payload: []event.StreamCreateRequestPayload{task},
		})
		t.currentTasks[task.LiveID] = task
	}

	// 5. 执行创建操作
	if len(tasksToCreate) > 0 {
		t.eventBus.Publish(event.Event{
			Type:    event.StreamCreateRequest,
			Payload: tasksToCreate,
		})
		// 更新当前任务缓存
		for _, task := range tasksToCreate {
			t.currentTasks[task.LiveID] = task
		}
	}

	logger.Info("Task changes processed",
		"deleted", len(tasksToDelete),
		"created", len(tasksToCreate),
		"updated", len(tasksToUpdate),
		"total_current", len(t.currentTasks))
}
