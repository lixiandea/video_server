package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	apperrors "github.com/lixiandea/video_server/pkg/errors"
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

// transcodeConsumerGroup 是转码队列唯一的消费者组名，读消息、确认与回收共用。
const transcodeConsumerGroup = "transcode_workers"

// TranscodeTask 转码任务
type TranscodeTask struct {
	TaskID       string     `json:"task_id"`
	VideoID      string     `json:"video_id"`
	SourcePath   string     `json:"source_path"`
	TargetFormat string     `json:"target_format"`
	Quality      string     `json:"quality"` // low, medium, high
	Status       TaskStatus `json:"status"`
	Progress     int        `json:"progress"` // 0-100
	Retries      int        `json:"retries,omitempty"` // 已重试次数
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
	client   *redis.Client
	config   QueueConfig
	consumer string
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

// consumerName 返回本进程固定的消费者名（含 PID）。
// 必须固定：若每次读取都用新名字，消息会散落到无数消费者名下，pending 永远无法回收。
func (q *TaskQueue) consumerName() string {
	if q.consumer == "" {
		q.consumer = fmt.Sprintf("worker-%d", os.Getpid())
	}
	return q.consumer
}

// EnqueueTranscodeTask 添加转码任务到队列
func (q *TaskQueue) EnqueueTranscodeTask(ctx context.Context, task *TranscodeTask) error {
	if task.TaskID == "" {
		task.TaskID = uuid.New().String()
	}
	task.Status = TaskStatusPending
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

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
			"retries":      task.Retries,
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
	q.client.XGroupCreateMkStream(ctx, q.config.QueueName, transcodeConsumerGroup, "0")

	// 使用 XREADGROUP 消费消息组
	result, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    transcodeConsumerGroup,
		Consumer: q.consumerName(),
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

// ReclaimStaleTasks 认领空闲超过 minIdle 的未确认消息（worker 崩溃、处理超时等场景），
// 返回需要重新处理的任务。没有该机制时这些消息会永远停留在 pending 列表里，
// 既不会被重投也不会被统计为失败。
//
// 这里刻意不用 XAUTOCLAIM：go-redis v8.11.5 的 XAutoClaim 解析器要求 2 元素回复，
// 而 Redis 7 返回 3 元素（额外带已删除消息 ID），会直接报 "got 3, wanted 2"，
// 因此改为 XPENDING 找超时消息 + XCLAIM 接管。
func (q *TaskQueue) ReclaimStaleTasks(ctx context.Context, minIdle time.Duration, count int64) ([]*TranscodeTask, error) {
	q.client.XGroupCreateMkStream(ctx, q.config.QueueName, transcodeConsumerGroup, "0")

	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.config.QueueName,
		Group:  transcodeConsumerGroup,
		Idle:   minIdle,
		Start:  "-",
		End:    "+",
		Count:  count,
	}).Result()
	if err != nil {
		if err == redis.Nil || isNoGroupError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list pending tasks: %w", err)
	}

	tasks := make([]*TranscodeTask, 0, len(pending))
	for _, p := range pending {
		if p.Idle < minIdle {
			continue
		}

		messages, err := q.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   q.config.QueueName,
			Group:    transcodeConsumerGroup,
			Consumer: q.consumerName(),
			MinIdle:  minIdle,
			Messages: []string{p.ID},
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return tasks, fmt.Errorf("failed to claim pending task %s: %w", p.ID, err)
		}

		for _, msg := range messages {
			task, ok := q.parseTaskMessage(ctx, msg)
			if !ok {
				continue
			}
			task.MessageID = msg.ID
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// parseTaskMessage 从流消息还原任务；无法解析的消息直接确认丢弃，避免被反复回收。
func (q *TaskQueue) parseTaskMessage(ctx context.Context, msg redis.XMessage) (*TranscodeTask, bool) {
	data, ok := msg.Values["data"].(string)
	if !ok {
		logging.GetLogger().Warn("Message without task data", zap.String("message_id", msg.ID))
		q.AcknowledgeTask(ctx, q.config.QueueName, msg.ID)
		return nil, false
	}

	var task TranscodeTask
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		logging.GetLogger().Warn("Failed to unmarshal task message",
			zap.String("message_id", msg.ID),
			zap.Error(err))
		q.AcknowledgeTask(ctx, q.config.QueueName, msg.ID)
		return nil, false
	}

	return &task, true
}

// AcknowledgeTask 确认任务完成
func (q *TaskQueue) AcknowledgeTask(ctx context.Context, stream, messageID string) error {
	return q.client.XAck(ctx, stream, transcodeConsumerGroup, messageID).Err()
}

// GetTaskStatus 获取任务状态
func (q *TaskQueue) GetTaskStatus(ctx context.Context, taskID string) (*TranscodeTask, error) {
	taskKey := fmt.Sprintf("task:%s", taskID)
	data, err := q.client.Get(ctx, taskKey).Bytes()
	if err == redis.Nil {
		return nil, apperrors.NotFoundf("task not found")
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

	// 获取待处理消息数；消费者组尚未创建（worker 未启动过）时视为 0
	pendingCount := 0
	pending, err := q.client.XPending(ctx, q.config.QueueName, transcodeConsumerGroup).Result()
	if err != nil && err != redis.Nil && !isNoGroupError(err) {
		return nil, err
	}
	if pending != nil {
		pendingCount = int(pending.Count)
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
		PendingCount: pendingCount,
		ProcessingCount: processing,
		CompletedCount: completed,
		FailedCount:  failed,
		TotalTasks:   len(keys),
	}, nil
}

// isNoGroupError 判断 Redis 返回的是否为“消费者组不存在”(NOGROUP) 错误。
// 该错误在转码 worker 从未消费过队列时属于正常状态，不应作为失败上报。
func isNoGroupError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NOGROUP")
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
