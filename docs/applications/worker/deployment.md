# 后台工作服务 (Worker) 部署指南

## 概述

Worker是视频服务的异步任务处理器，负责处理耗时操作如视频转码、文件上传、通知发送等后台任务。

## 架构设计

### 核心组件

1. **任务队列**: Redis队列存储待处理任务
2. **工作者池**: 多个工作协程并发处理任务
3. **任务处理器**: 具体业务逻辑实现
4. **状态管理**: 任务状态跟踪和持久化

### 工作流程

```
任务生产者 → Redis队列 → Worker消费者 → 任务处理器 → 结果存储
     ↓                                    ↓
   API Server                         数据库/存储
```

## 部署配置

### Docker部署

#### Dockerfile

```dockerfile
# Dockerfile.worker
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata ffmpeg

WORKDIR /app

COPY --from=builder /app/worker .
COPY .env .

RUN mkdir -p logs storage/temp

CMD ["./worker"]
```

#### Docker Compose配置

```yaml
version: '3.8'

services:
  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    container_name: video_server_worker
    restart: always
    environment:
      - WORKER_CONCURRENCY=5
      - WORKER_QUEUE_NAME=video_tasks
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=video_user
      - DB_PASSWORD=video_password
      - DB_NAME=video_server
      - REDIS_ADDR=redis:6379
      - STORAGE_PATH=./storage
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_started
    volumes:
      - ./storage:/app/storage
      - ./logs:/app/logs
    networks:
      - video_network
```

## 配置管理

### 配置文件

```yaml
# config-worker.yaml
worker:
  # 工作者配置
  concurrency: 5           # 并发工作者数量
  queue_name: "video_tasks" # 队列名称
  prefetch_count: 10       # 预取任务数量
  graceful_shutdown: 30    # 优雅关闭超时(秒)
  
  # 任务超时配置
  task_timeout: 3600       # 任务执行超时(秒)
  heartbeat_interval: 30   # 心跳间隔(秒)
  
  # 重试配置
  max_retries: 3           # 最大重试次数
  retry_delay: 60          # 重试延迟(秒)
  exponential_backoff: true # 指数退避

database:
  host: "localhost"
  port: 3306
  user: "video_user"
  password: "video_password"
  name: "video_server"
  max_idle_conns: 5
  max_open_conns: 20

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10

storage:
  base_path: "./storage"
  temp_path: "./storage/temp"
  video_path: "./storage/videos"
  max_file_size: 1073741824  # 1GB

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "./logs/worker.log"
  max_size: 50
  max_age: 30
```

## 核心实现

### 任务队列管理

```go
package worker

import (
    "context"
    "encoding/json"
    "time"
    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
)

type TaskQueue struct {
    client    *redis.Client
    queueName string
    logger    *zap.Logger
}

type Task struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Payload     map[string]interface{} `json:"payload"`
    Priority    int                    `json:"priority"`
    MaxRetries  int                    `json:"max_retries"`
    RetryCount  int                    `json:"retry_count"`
    CreatedAt   time.Time              `json:"created_at"`
    Processing  bool                   `json:"processing"`
    ProcessedBy string                 `json:"processed_by"`
}

func NewTaskQueue(client *redis.Client, queueName string, logger *zap.Logger) *TaskQueue {
    return &TaskQueue{
        client:    client,
        queueName: queueName,
        logger:    logger,
    }
}

func (tq *TaskQueue) Enqueue(task *Task) error {
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }
    
    // 使用有序集合按优先级排序
    score := float64(task.Priority) + float64(time.Now().UnixNano())/1e15
    return tq.client.ZAdd(context.Background(), tq.queueName, &redis.Z{
        Score:  score,
        Member: data,
    }).Err()
}

func (tq *TaskQueue) Dequeue() (*Task, error) {
    // 原子性地获取并移除最高优先级任务
    result := tq.client.ZPopMin(context.Background(), tq.queueName, 1)
    if result.Err() != nil {
        return nil, result.Err()
    }
    
    members := result.Val()
    if len(members) == 0 {
        return nil, redis.Nil
    }
    
    var task Task
    if err := json.Unmarshal([]byte(members[0].Member.(string)), &task); err != nil {
        return nil, err
    }
    
    return &task, nil
}

func (tq *TaskQueue) Requeue(task *Task, delay time.Duration) error {
    task.RetryCount++
    task.Processing = false
    task.ProcessedBy = ""
    
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }
    
    // 延迟重入队列
    score := float64(time.Now().Add(delay).UnixNano()) / 1e9
    return tq.client.ZAdd(context.Background(), tq.queueName, &redis.Z{
        Score:  score,
        Member: data,
    }).Err()
}
```

### 工作者池

```go
type WorkerPool struct {
    workers     []*Worker
    taskQueue   *TaskQueue
    taskHandler TaskHandler
    semaphore   chan struct{}
    logger      *zap.Logger
    ctx         context.Context
    cancel      context.CancelFunc
}

type Worker struct {
    id       int
    pool     *WorkerPool
    logger   *zap.Logger
}

func NewWorkerPool(concurrency int, queue *TaskQueue, handler TaskHandler, logger *zap.Logger) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    
    wp := &WorkerPool{
        workers:     make([]*Worker, concurrency),
        taskQueue:   queue,
        taskHandler: handler,
        semaphore:   make(chan struct{}, concurrency),
        logger:      logger,
        ctx:         ctx,
        cancel:      cancel,
    }
    
    // 初始化工作者
    for i := 0; i < concurrency; i++ {
        wp.workers[i] = &Worker{
            id:     i + 1,
            pool:   wp,
            logger: logger.With(zap.Int("worker_id", i+1)),
        }
    }
    
    return wp
}

func (wp *WorkerPool) Start() {
    wp.logger.Info("Starting worker pool", zap.Int("concurrency", len(wp.workers)))
    
    for _, worker := range wp.workers {
        go worker.Run()
    }
}

func (wp *WorkerPool) Stop() {
    wp.logger.Info("Stopping worker pool")
    wp.cancel()
    
    // 等待所有工作者完成
    for i := 0; i < cap(wp.semaphore); i++ {
        wp.semaphore <- struct{}{}
    }
}

func (w *Worker) Run() {
    w.logger.Info("Worker started")
    
    for {
        select {
        case <-w.pool.ctx.Done():
            w.logger.Info("Worker stopping")
            return
        default:
            w.processTask()
        }
    }
}

func (w *Worker) processTask() {
    // 获取任务
    task, err := w.pool.taskQueue.Dequeue()
    if err != nil {
        if err != redis.Nil {
            w.logger.Error("Failed to dequeue task", zap.Error(err))
        }
        time.Sleep(1 * time.Second) // 避免忙等待
        return
    }
    
    // 标记任务为处理中
    task.Processing = true
    task.ProcessedBy = fmt.Sprintf("worker-%d", w.id)
    
    w.logger.Info("Processing task", 
        zap.String("task_id", task.ID),
        zap.String("task_type", task.Type))
    
    // 执行任务
    ctx, cancel := context.WithTimeout(w.pool.ctx, time.Hour)
    defer cancel()
    
    startTime := time.Now()
    err = w.pool.taskHandler.HandleTask(ctx, task)
    duration := time.Since(startTime)
    
    if err != nil {
        w.handleTaskFailure(task, err)
    } else {
        w.handleTaskSuccess(task, duration)
    }
}
```

### 任务处理器

```go
type TaskHandler interface {
    HandleTask(ctx context.Context, task *Task) error
    SupportedTaskTypes() []string
}

// 视频处理任务处理器
type VideoProcessor struct {
    storage *storage.StorageService
    db      *gorm.DB
    logger  *zap.Logger
}

func (vp *VideoProcessor) SupportedTaskTypes() []string {
    return []string{"video_upload", "video_transcode", "video_thumbnail"}
}

func (vp *VideoProcessor) HandleTask(ctx context.Context, task *Task) error {
    switch task.Type {
    case "video_upload":
        return vp.handleVideoUpload(ctx, task)
    case "video_transcode":
        return vp.handleVideoTranscode(ctx, task)
    case "video_thumbnail":
        return vp.handleVideoThumbnail(ctx, task)
    default:
        return fmt.Errorf("unsupported task type: %s", task.Type)
    }
}

func (vp *VideoProcessor) handleVideoUpload(ctx context.Context, task *Task) error {
    filePath := task.Payload["file_path"].(string)
    videoID := task.Payload["video_id"].(string)
    
    // 验证文件
    if err := vp.validateVideoFile(filePath); err != nil {
        return err
    }
    
    // 生成缩略图
    thumbnailPath, err := vp.generateThumbnail(filePath)
    if err != nil {
        vp.logger.Warn("Failed to generate thumbnail", zap.Error(err))
    }
    
    // 更新数据库
    return vp.db.Model(&Video{}).Where("id = ?", videoID).Updates(map[string]interface{}{
        "status":         "processed",
        "thumbnail_url":  thumbnailPath,
        "processed_at":   time.Now(),
    }).Error
}

func (vp *VideoProcessor) handleVideoTranscode(ctx context.Context, task *Task) error {
    inputPath := task.Payload["input_path"].(string)
    outputPath := task.Payload["output_path"].(string)
    format := task.Payload["format"].(string)
    
    // 执行转码
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-i", inputPath,
        "-c:v", "libx264",
        "-c:a", "aac",
        "-preset", "medium",
        "-crf", "23",
        outputPath,
    )
    
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("transcoding failed: %w", err)
    }
    
    // 更新任务状态
    return vp.updateTranscodeStatus(task.Payload["task_id"].(string), "completed", outputPath)
}

func (vp *VideoProcessor) handleVideoThumbnail(ctx context.Context, task *Task) error {
    videoPath := task.Payload["video_path"].(string)
    outputPath := task.Payload["output_path"].(string)
    timestamp := task.Payload["timestamp"].(float64)
    
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-i", videoPath,
        "-ss", fmt.Sprintf("%.2f", timestamp),
        "-vframes", "1",
        "-f", "image2",
        outputPath,
    )
    
    return cmd.Run()
}
```

## 任务管理

### 任务状态跟踪

```go
type TaskStatus struct {
    TaskID      string    `json:"task_id"`
    Status      string    `json:"status"`  // pending, processing, completed, failed
    WorkerID    string    `json:"worker_id,omitempty"`
    StartedAt   time.Time `json:"started_at,omitempty"`
    CompletedAt time.Time `json:"completed_at,omitempty"`
    Error       string    `json:"error,omitempty"`
    Progress    float64   `json:"progress,omitempty"`
}

type TaskTracker struct {
    client *redis.Client
    logger *zap.Logger
}

func (tt *TaskTracker) UpdateStatus(taskID string, status *TaskStatus) error {
    key := fmt.Sprintf("task_status:%s", taskID)
    data, err := json.Marshal(status)
    if err != nil {
        return err
    }
    
    return tt.client.Set(context.Background(), key, data, 24*time.Hour).Err()
}

func (tt *TaskTracker) GetStatus(taskID string) (*TaskStatus, error) {
    key := fmt.Sprintf("task_status:%s", taskID)
    data, err := tt.client.Get(context.Background(), key).Bytes()
    if err != nil {
        return nil, err
    }
    
    var status TaskStatus
    return &status, json.Unmarshal(data, &status)
}

func (tt *TaskTracker) UpdateProgress(taskID string, progress float64) error {
    status, err := tt.GetStatus(taskID)
    if err != nil {
        return err
    }
    
    status.Progress = progress
    status.Status = "processing"
    return tt.UpdateStatus(taskID, status)
}
```

### 任务API接口

```go
func setupTaskRoutes(r *gin.Engine, workerPool *WorkerPool, taskQueue *TaskQueue) {
    tasks := r.Group("/api/v1/tasks")
    {
        tasks.POST("/", enqueueTask)
        tasks.GET("/:task_id", getTaskStatus)
        tasks.GET("/", listTasks)
        tasks.DELETE("/:task_id", cancelTask)
    }
}

type EnqueueTaskRequest struct {
    Type     string                 `json:"type" binding:"required"`
    Payload  map[string]interface{} `json:"payload" binding:"required"`
    Priority int                    `json:"priority"`
}

func enqueueTask(c *gin.Context) {
    var req EnqueueTaskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    task := &Task{
        ID:         generateTaskID(),
        Type:       req.Type,
        Payload:    req.Payload,
        Priority:   req.Priority,
        MaxRetries: 3,
        CreatedAt:  time.Now(),
    }
    
    if err := taskQueue.Enqueue(task); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusAccepted, gin.H{
        "task_id": task.ID,
        "status":  "queued",
    })
}

func getTaskStatus(c *gin.Context) {
    taskID := c.Param("task_id")
    status, err := taskTracker.GetStatus(taskID)
    if err != nil {
        if err == redis.Nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        }
        return
    }
    
    c.JSON(http.StatusOK, status)
}
```

## 监控和指标

### Prometheus指标

```go
var (
    tasksProcessedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "worker_tasks_processed_total",
            Help: "Total number of tasks processed",
        },
        []string{"task_type", "status"},
    )
    
    taskProcessingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "worker_task_processing_duration_seconds",
            Help:    "Task processing duration in seconds",
            Buckets: []float64{1, 10, 30, 60, 120, 300, 600, 1800},
        },
        []string{"task_type"},
    )
    
    queueLength = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "worker_queue_length",
            Help: "Current queue length",
        },
        []string{"queue_name"},
    )
    
    activeWorkers = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "worker_active_workers",
            Help: "Number of active workers",
        },
    )
)

func init() {
    prometheus.MustRegister(tasksProcessedTotal)
    prometheus.MustRegister(taskProcessingDuration)
    prometheus.MustRegister(queueLength)
    prometheus.MustRegister(activeWorkers)
}

// 在任务处理中记录指标
func (w *Worker) handleTaskSuccess(task *Task, duration time.Duration) {
    tasksProcessedTotal.WithLabelValues(task.Type, "success").Inc()
    taskProcessingDuration.WithLabelValues(task.Type).Observe(duration.Seconds())
    
    w.logger.Info("Task completed successfully",
        zap.String("task_id", task.ID),
        zap.String("task_type", task.Type),
        zap.Duration("duration", duration))
}

func (w *Worker) handleTaskFailure(task *Task, err error) {
    tasksProcessedTotal.WithLabelValues(task.Type, "failed").Inc()
    
    w.logger.Error("Task failed",
        zap.String("task_id", task.ID),
        zap.String("task_type", task.Type),
        zap.Error(err))
    
    // 处理重试逻辑
    if task.RetryCount < task.MaxRetries {
        delay := calculateRetryDelay(task.RetryCount)
        w.pool.taskQueue.Requeue(task, delay)
        w.logger.Info("Task requeued for retry",
            zap.String("task_id", task.ID),
            zap.Int("retry_count", task.RetryCount),
            zap.Duration("delay", delay))
    } else {
        w.logger.Error("Task failed permanently",
            zap.String("task_id", task.ID),
            zap.Int("max_retries", task.MaxRetries))
    }
}
```

## 错误处理和重试

```go
func (w *Worker) handleTaskFailure(task *Task, err error) {
    // 记录错误日志
    w.logger.Error("Task processing failed",
        zap.String("task_id", task.ID),
        zap.String("task_type", task.Type),
        zap.Error(err),
        zap.Int("retry_count", task.RetryCount))
    
    // 更新任务状态
    status := &TaskStatus{
        TaskID:      task.ID,
        Status:      "failed",
        Error:       err.Error(),
        CompletedAt: time.Now(),
    }
    taskTracker.UpdateStatus(task.ID, status)
    
    // 决定是否重试
    if task.RetryCount < task.MaxRetries {
        w.scheduleRetry(task, err)
    } else {
        // 达到最大重试次数，发送告警
        w.sendFailureAlert(task, err)
    }
}

func (w *Worker) scheduleRetry(task *Task, err error) {
    task.RetryCount++
    
    // 计算延迟时间（指数退避）
    delay := calculateExponentialBackoff(task.RetryCount)
    
    // 重新入队
    go func() {
        time.Sleep(delay)
        if err := w.pool.taskQueue.Requeue(task, 0); err != nil {
            w.logger.Error("Failed to requeue task", zap.Error(err))
        }
    }()
    
    w.logger.Info("Scheduled task retry",
        zap.String("task_id", task.ID),
        zap.Int("retry_count", task.RetryCount),
        zap.Duration("delay", delay))
}

func calculateExponentialBackoff(attempt int) time.Duration {
    baseDelay := 1 * time.Minute
    maxDelay := 30 * time.Minute
    
    delay := baseDelay * time.Duration(1<<uint(attempt-1))
    if delay > maxDelay {
        return maxDelay
    }
    return delay
}
```

## 健康检查

```go
func setupHealthRoutes(r *gin.Engine, workerPool *WorkerPool, taskQueue *TaskQueue) {
    r.GET("/health", func(c *gin.Context) {
        // 检查队列长度
        queueLen, err := taskQueue.GetLength()
        if err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "status": "unhealthy",
                "error":  err.Error(),
            })
            return
        }
        
        c.JSON(http.StatusOK, gin.H{
            "status":      "healthy",
            "service":     "worker",
            "queue_length": queueLen,
            "active_workers": len(workerPool.workers),
            "version":     "1.0.0",
        })
    })
    
    r.GET("/health/detail", func(c *gin.Context) {
        queueLen, _ := taskQueue.GetLength()
        processingTasks, _ := getProcessingTasks()
        
        c.JSON(http.StatusOK, gin.H{
            "status":           "healthy",
            "queue_length":     queueLen,
            "processing_tasks": processingTasks,
            "worker_stats":     getWorkerStats(),
            "system_metrics":   getSystemMetrics(),
        })
    })
}

func getProcessingTasks() []TaskStatus {
    // 从Redis获取正在处理的任务
    // 实现细节省略
    return []TaskStatus{}
}

func getWorkerStats() map[string]interface{} {
    // 获取工作者统计信息
    // 实现细节省略
    return map[string]interface{}{}
}
```

## 部署检查清单

- [ ] Redis连接正常
- [ ] 数据库连接正常
- [ ] 存储目录权限正确
- [ ] FFmpeg等依赖工具已安装
- [ ] 并发配置合理
- [ ] 任务超时设置适当
- [ ] 重试机制配置正确
- [ ] 监控指标正常暴露
- [ ] 健康检查端点可访问
- [ ] 日志配置正确

## 下一步

- [前端部署指南](../frontend/deployment.md)
- [API Server配置](../api-server/deployment.md)
- [基础设施监控](../../infrastructure/monitoring/README.md)