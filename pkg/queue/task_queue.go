package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/lixiandea/video_server/pkg/logging"
	"go.uber.org/zap"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeTranscode TaskType = "transcode"
	TaskTypeThumbnail TaskType = "thumbnail"
	TaskTypeHLS       TaskType = "hls"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TranscodeTask 转码任务
type TranscodeTask struct {
	TaskID       string     `json:"task_id"`
	VideoID      string     `json:"video_id"`
	SourcePath   string     `json:"source_path"`
	TargetFormat string     `json:"target_format"`
	Quality      string     `json:"quality"` // low, medium, high
	Status       TaskStatus `json:"status"`
	Progress     int        `json:"progress"` // 0-100
	Error        string     `json:"error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	MessageID    string     `json:"-"` // Redis 消息 ID，不序列化
}

// QueueConfig 队列配置
type QueueConfig struct {
	QueueName     string
	MaxRetries    int
	VisibilityTimeout time.Duration
}

// DefaultQueueConfig 默认队列配置
var DefaultQueueConfig = QueueConfig{
	QueueName:         "video_transcode_queue",
	MaxRetries:        3,
	VisibilityTimeout: 30 * time.Minute,
}

// TaskQueue 任务队列接口
type TaskQueue struct {
	client *redis.Client
	config QueueConfig
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(client *redis.Client, config QueueConfig) *TaskQueue {
	if config.QueueName == "" {
		config = DefaultQueueConfig
	}
	return &TaskQueue{
		client: client,
		config: config,
	}
}

// EnqueueTranscodeTask 添加转码任务到队列
func (q *TaskQueue) EnqueueTranscodeTask(ctx context.Context, task *TranscodeTask) error {
	if task.TaskID == "" {
		task.TaskID = uuid.New().String()
	}
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	data, err := json.Marshal(task)
	if err != nil {
		logging.GetLogger().Error("Failed to marshal task", zap.Error(err))
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 添加到 Redis Stream
	err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.config.QueueName,
		Values: map[string]interface{}{
			"task_id":      task.TaskID,
			"data":         string(data),
			"status":       string(task.Status),
			"retries":      0,
			"created_at":   task.CreatedAt.Format(time.RFC3339),
		},
	}).Err()

	if err != nil {
		logging.GetLogger().Error("Failed to enqueue task", zap.Error(err))
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	// 更新任务状态到 Redis Hash
	taskKey := fmt.Sprintf("task:%s", task.TaskID)
	taskData, _ := json.Marshal(task)
	err = q.client.Set(ctx, taskKey, taskData, 24*time.Hour).Err()
	if err != nil {
		logging.GetLogger().Warn("Failed to cache task status", zap.Error(err))
	}

	logging.GetLogger().Info("Transcode task enqueued",
		zap.String("task_id", task.TaskID),
		zap.String("video_id", task.VideoID))

	return nil
}

// DequeueTranscodeTask 从队列获取任务
func (q *TaskQueue) DequeueTranscodeTask(ctx context.Context) (*TranscodeTask, error) {
	// 首先尝试创建消费者组（如果不存在）
	q.client.XGroupCreateMkStream(ctx, q.config.QueueName, "transcode_workers", "0")

	// 使用 XREADGROUP 消费消息组
	result, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    "transcode_workers",
		Consumer: fmt.Sprintf("worker-%s", uuid.New().String()),
		Streams:  []string{q.config.QueueName, ">"},
		Count:    1,
		Block:    5 * time.Second,
	}).Result()

	if err == redis.Nil {
		return nil, nil // 队列为空
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue task: %w", err)
	}

	if len(result) == 0 || len(result[0].Messages) == 0 {
		return nil, nil
	}

	msg := result[0].Messages[0]
	data := msg.Values["data"].(string)

	var task TranscodeTask
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// 更新任务状态为 processing
	now := time.Now()
	task.Status = TaskStatusProcessing
	task.StartedAt = &now
	task.MessageID = msg.ID // 保存消息 ID 用于确认

	return &task, nil
}

// AcknowledgeTask 确认任务完成
func (q *TaskQueue) AcknowledgeTask(ctx context.Context, stream, messageID string) error {
	return q.client.XAck(ctx, stream, "transcode_workers", messageID).Err()
}

// GetTaskStatus 获取任务状态
func (q *TaskQueue) GetTaskStatus(ctx context.Context, taskID string) (*TranscodeTask, error) {
	taskKey := fmt.Sprintf("task:%s", taskID)
	data, err := q.client.Get(ctx, taskKey).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task TranscodeTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// UpdateTaskStatus 更新任务状态
func (q *TaskQueue) UpdateTaskStatus(ctx context.Context, task *TranscodeTask) error {
	taskKey := fmt.Sprintf("task:%s", task.TaskID)
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = q.client.Set(ctx, taskKey, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// GetQueueStats 获取队列统计信息
func (q *TaskQueue) GetQueueStats(ctx context.Context) (*QueueStats, error) {
	// 获取队列长度
	length, err := q.client.XLen(ctx, q.config.QueueName).Result()
	if err != nil {
		return nil, err
	}

	// 获取待处理消息数
	pending, err := q.client.XPending(ctx, q.config.QueueName, "transcode_workers").Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// 获取所有任务键
	keys, err := q.client.Keys(ctx, "task:*").Result()
	if err != nil {
		return nil, err
	}

	// 统计各状态任务数
	var completed, failed, processing int
	for _, key := range keys {
		data, err := q.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var task TranscodeTask
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		switch task.Status {
		case TaskStatusCompleted:
			completed++
		case TaskStatusFailed:
			failed++
		case TaskStatusProcessing:
			processing++
		}
	}

	return &QueueStats{
		QueueLength:  int(length),
		PendingCount: int(pending.Count),
		ProcessingCount: processing,
		CompletedCount: completed,
		FailedCount:  failed,
		TotalTasks:   len(keys),
	}, nil
}

// QueueStats 队列统计信息
type QueueStats struct {
	QueueLength     int `json:"queue_length"`
	PendingCount    int `json:"pending_count"`
	ProcessingCount int `json:"processing_count"`
	CompletedCount  int `json:"completed_count"`
	FailedCount     int `json:"failed_count"`
	TotalTasks      int `json:"total_tasks"`
}

// CleanupOldTasks 清理旧任务
func (q *TaskQueue) CleanupOldTasks(ctx context.Context, maxAge time.Duration) (int, error) {
	keys, err := q.client.Keys(ctx, "task:*").Result()
	if err != nil {
		return 0, err
	}

	deleted := 0
	cutoff := time.Now().Add(-maxAge)

	for _, key := range keys {
		data, err := q.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var task TranscodeTask
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
			if err := q.client.Del(ctx, key).Err(); err == nil {
				deleted++
			}
		}
	}

	return deleted, nil
}

// GetQueueName 获取队列名称
func (q *TaskQueue) GetQueueName() string {
	return q.config.QueueName
}
