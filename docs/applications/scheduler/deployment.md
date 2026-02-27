# 任务调度服务 (Scheduler) 部署指南

## 概述

Scheduler是视频服务的后台任务调度器，负责执行定时任务、清理过期数据、生成报表等周期性工作。

## 架构设计

### 核心功能

1. **定时任务调度**: 基于cron表达式的任务调度
2. **任务管理**: 任务的创建、修改、删除和监控
3. **执行引擎**: 任务执行和状态跟踪
4. **错误处理**: 失败任务的重试和告警

### 任务类型

- **清理任务**: 清理过期视频、临时文件
- **统计任务**: 生成用户、视频统计数据
- **通知任务**: 发送系统通知和提醒
- **备份任务**: 数据库和文件备份

## 部署配置

### Docker部署

#### Dockerfile

```dockerfile
# Dockerfile.scheduler
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scheduler ./cmd/scheduler

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/scheduler .
COPY config-scheduler.yaml .
COPY .env .

RUN mkdir -p logs

EXPOSE 8089

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8089/health || exit 1

CMD ["./scheduler"]
```

#### Docker Compose配置

```yaml
# docker-compose.yml
version: '3.8'

services:
  scheduler:
    build:
      context: .
      dockerfile: Dockerfile.scheduler
    container_name: video_server_scheduler
    restart: always
    ports:
      - "8089:8089"
    environment:
      - SCHEDULER_PORT=8089
      - SCHEDULER_MODE=release
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=video_user
      - DB_PASSWORD=video_password
      - DB_NAME=video_server
      - LOG_LEVEL=info
    depends_on:
      mysql:
        condition: service_healthy
    volumes:
      - ./logs:/app/logs
    networks:
      - video_network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8089/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
```

## 配置管理

### 主配置文件

```yaml
# config-scheduler.yaml
server:
  port: "8089"
  mode: "release"  # debug/release/test
  graceful_shutdown_timeout: 30

database:
  host: "localhost"
  port: 3306
  user: "video_user"
  password: "video_password"
  name: "video_server"
  charset: "utf8mb4"
  max_idle_conns: 5
  max_open_conns: 20
  conn_max_lifetime: 1800

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "./logs/scheduler.log"
  max_size: 50
  max_age: 30
  max_backups: 5

scheduler:
  # 任务执行配置
  max_concurrent_jobs: 10
  job_timeout: 3600  # 1小时
  retry_attempts: 3
  retry_delay: 60    # 1分钟
  
  # 任务扫描间隔
  scan_interval: 10  # 10秒
  
  # 任务历史保留天数
  history_retention_days: 30

jobs:
  # 视频清理任务
  - name: "video_cleanup"
    schedule: "0 2 * * *"  # 每天凌晨2点
    handler: "VideoCleanupJob"
    enabled: true
    config:
      retention_days: 30
      batch_size: 100
      
  # 用户统计任务
  - name: "user_statistics"
    schedule: "0 1 * * *"  # 每天凌晨1点
    handler: "UserStatisticsJob"
    enabled: true
    config:
      report_format: "json"
      
  # 系统监控任务
  - name: "system_monitoring"
    schedule: "*/5 * * * *"  # 每5分钟
    handler: "SystemMonitoringJob"
    enabled: true
    config:
      cpu_threshold: 80
      memory_threshold: 85
```

### 任务配置

```go
// 任务定义结构
type JobConfig struct {
    Name     string         `yaml:"name"`
    Schedule string         `yaml:"schedule"`  // cron表达式
    Handler  string         `yaml:"handler"`
    Enabled  bool           `yaml:"enabled"`
    Config   map[string]interface{} `yaml:"config"`
}

// 任务执行器接口
type JobHandler interface {
    Name() string
    Execute(ctx context.Context, config map[string]interface{}) error
    ValidateConfig(config map[string]interface{}) error
}
```

## 核心实现

### 任务调度引擎

```go
package scheduler

import (
    "context"
    "time"
    "github.com/robfig/cron/v3"
    "go.uber.org/zap"
)

type Scheduler struct {
    cron       *cron.Cron
    jobStore   JobStore
    executor   JobExecutor
    logger     *zap.Logger
    config     *Config
}

func NewScheduler(config *Config, logger *zap.Logger) *Scheduler {
    s := &Scheduler{
        cron:     cron.New(cron.WithSeconds()),
        jobStore: NewJobStore(config.Database),
        executor: NewJobExecutor(config.Scheduler),
        logger:   logger,
        config:   config,
    }
    
    return s
}

func (s *Scheduler) Start() error {
    // 加载所有启用的任务
    jobs, err := s.jobStore.ListEnabledJobs()
    if err != nil {
        return err
    }
    
    // 注册任务到cron调度器
    for _, job := range jobs {
        if err := s.registerJob(job); err != nil {
            s.logger.Error("Failed to register job", 
                zap.String("job", job.Name), 
                zap.Error(err))
            continue
        }
    }
    
    // 启动cron调度器
    s.cron.Start()
    s.logger.Info("Scheduler started")
    return nil
}

func (s *Scheduler) registerJob(job *Job) error {
    entryID, err := s.cron.AddFunc(job.Schedule, func() {
        s.executeJob(job)
    })
    
    if err != nil {
        return err
    }
    
    job.EntryID = entryID
    return nil
}

func (s *Scheduler) executeJob(job *Job) {
    ctx, cancel := context.WithTimeout(context.Background(), 
        time.Duration(s.config.Scheduler.JobTimeout)*time.Second)
    defer cancel()
    
    startTime := time.Now()
    s.logger.Info("Executing job", zap.String("job", job.Name))
    
    // 记录执行开始
    execution := &JobExecution{
        JobName:   job.Name,
        StartTime: startTime,
        Status:    "running",
    }
    
    if err := s.jobStore.CreateExecution(execution); err != nil {
        s.logger.Error("Failed to create execution record", zap.Error(err))
        return
    }
    
    // 执行任务
    handler := s.getJobHandler(job.Handler)
    if handler == nil {
        s.markExecutionFailed(execution, "Unknown job handler")
        return
    }
    
    err := s.executor.Execute(ctx, handler, job.Config)
    
    // 更新执行结果
    if err != nil {
        s.markExecutionFailed(execution, err.Error())
        s.handleJobFailure(job, err)
    } else {
        s.markExecutionSuccess(execution)
        s.logger.Info("Job executed successfully", 
            zap.String("job", job.Name),
            zap.Duration("duration", time.Since(startTime)))
    }
}
```

### 任务执行器

```go
type JobExecutor struct {
    maxConcurrent int
    semaphore     chan struct{}
    logger        *zap.Logger
}

func NewJobExecutor(config SchedulerConfig) *JobExecutor {
    return &JobExecutor{
        maxConcurrent: config.MaxConcurrentJobs,
        semaphore:     make(chan struct{}, config.MaxConcurrentJobs),
        logger:        zap.L(),
    }
}

func (je *JobExecutor) Execute(ctx context.Context, handler JobHandler, config map[string]interface{}) error {
    // 获取执行许可
    select {
    case je.semaphore <- struct{}{}:
        defer func() { <-je.semaphore }()
    case <-ctx.Done():
        return ctx.Err()
    }
    
    // 验证配置
    if err := handler.ValidateConfig(config); err != nil {
        return err
    }
    
    // 执行任务
    return handler.Execute(ctx, config)
}
```

### 具体任务实现

```go
// 视频清理任务
type VideoCleanupJob struct {
    db     *gorm.DB
    logger *zap.Logger
}

func (j *VideoCleanupJob) Name() string {
    return "VideoCleanupJob"
}

func (j *VideoCleanupJob) ValidateConfig(config map[string]interface{}) error {
    if _, exists := config["retention_days"]; !exists {
        return errors.New("retention_days is required")
    }
    if _, exists := config["batch_size"]; !exists {
        return errors.New("batch_size is required")
    }
    return nil
}

func (j *VideoCleanupJob) Execute(ctx context.Context, config map[string]interface{}) error {
    retentionDays := int(config["retention_days"].(float64))
    batchSize := int(config["batch_size"].(float64))
    
    cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
    
    for {
        // 查找过期视频
        var videos []Video
        result := j.db.Where("created_at < ?", cutoffTime).
            Limit(batchSize).
            Find(&videos)
        
        if result.Error != nil {
            return result.Error
        }
        
        if len(videos) == 0 {
            break // 没有更多视频需要清理
        }
        
        // 删除视频文件和记录
        for _, video := range videos {
            if err := j.deleteVideo(video); err != nil {
                j.logger.Error("Failed to delete video", 
                    zap.String("video_id", video.ID), 
                    zap.Error(err))
                continue
            }
        }
        
        // 如果处理的数量小于批次大小，说明已经处理完所有数据
        if len(videos) < batchSize {
            break
        }
        
        // 避免过于频繁的数据库查询
        time.Sleep(100 * time.Millisecond)
    }
    
    return nil
}

func (j *VideoCleanupJob) deleteVideo(video Video) error {
    // 删除视频文件
    if err := os.Remove(video.FilePath); err != nil && !os.IsNotExist(err) {
        return err
    }
    
    // 删除数据库记录
    if err := j.db.Delete(&video).Error; err != nil {
        return err
    }
    
    // 记录删除操作
    return j.db.Create(&VideoDeletionRecord{
        VideoID:   video.ID,
        DeletedAt: time.Now(),
    }).Error
}
```

## 健康检查和监控

### 健康检查端点

```go
func setupHealthRoutes(r *gin.Engine, scheduler *Scheduler) {
    r.GET("/health", func(c *gin.Context) {
        status := "healthy"
        if !scheduler.IsRunning() {
            status = "unhealthy"
        }
        
        c.JSON(http.StatusOK, gin.H{
            "status":     status,
            "service":    "scheduler",
            "version":    "1.0.0",
            "jobs_count": scheduler.GetJobCount(),
            "uptime":     time.Since(startTime).String(),
        })
    })
    
    r.GET("/health/detail", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":           "healthy",
            "next_executions":  scheduler.GetNextExecutions(),
            "recent_failures":  scheduler.GetRecentFailures(10),
            "active_jobs":      scheduler.GetActiveJobs(),
            "system_metrics":   getSystemMetrics(),
        })
    })
}
```

### Prometheus指标

```go
var (
    jobsExecutedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "scheduler_jobs_executed_total",
            Help: "Total number of jobs executed",
        },
        []string{"job_name", "status"},
    )
    
    jobExecutionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "scheduler_job_execution_duration_seconds",
            Help:    "Job execution duration in seconds",
            Buckets: []float64{1, 10, 30, 60, 120, 300, 600},
        },
        []string{"job_name"},
    )
    
    activeJobs = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "scheduler_active_jobs",
            Help: "Number of currently active jobs",
        },
    )
)

func init() {
    prometheus.MustRegister(jobsExecutedTotal)
    prometheus.MustRegister(jobExecutionDuration)
    prometheus.MustRegister(activeJobs)
}

// 在任务执行中记录指标
func (s *Scheduler) executeJob(job *Job) {
    startTime := time.Now()
    activeJobs.Inc()
    defer activeJobs.Dec()
    
    // ... 执行逻辑 ...
    
    duration := time.Since(startTime).Seconds()
    jobExecutionDuration.WithLabelValues(job.Name).Observe(duration)
    
    if err != nil {
        jobsExecutedTotal.WithLabelValues(job.Name, "failed").Inc()
    } else {
        jobsExecutedTotal.WithLabelValues(job.Name, "success").Inc()
    }
}
```

## 任务管理API

```go
// 任务管理路由
func setupJobManagementRoutes(r *gin.Engine, scheduler *Scheduler) {
    jobs := r.Group("/api/v1/jobs")
    {
        jobs.GET("/", listJobs)
        jobs.POST("/", createJob)
        jobs.GET("/:name", getJob)
        jobs.PUT("/:name", updateJob)
        jobs.DELETE("/:name", deleteJob)
        jobs.POST("/:name/run", runJobManually)
        jobs.GET("/:name/executions", listJobExecutions)
    }
}

type JobRequest struct {
    Name     string                 `json:"name" binding:"required"`
    Schedule string                 `json:"schedule" binding:"required"`
    Handler  string                 `json:"handler" binding:"required"`
    Enabled  bool                   `json:"enabled"`
    Config   map[string]interface{} `json:"config"`
}

func createJob(c *gin.Context) {
    var req JobRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    job := &Job{
        Name:     req.Name,
        Schedule: req.Schedule,
        Handler:  req.Handler,
        Enabled:  req.Enabled,
        Config:   req.Config,
    }
    
    if err := scheduler.CreateJob(job); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, job)
}

func listJobs(c *gin.Context) {
    jobs, err := scheduler.ListJobs()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, jobs)
}
```

## 错误处理和重试

```go
type RetryPolicy struct {
    MaxAttempts int
    Delay       time.Duration
    Backoff     BackoffStrategy
}

type BackoffStrategy interface {
    NextDelay(attempt int) time.Duration
}

type ExponentialBackoff struct {
    baseDelay time.Duration
    maxDelay  time.Duration
}

func (eb *ExponentialBackoff) NextDelay(attempt int) time.Duration {
    delay := eb.baseDelay * time.Duration(1<<uint(attempt))
    if delay > eb.maxDelay {
        return eb.maxDelay
    }
    return delay
}

func (s *Scheduler) handleJobFailure(job *Job, err error) {
    // 记录失败
    s.logger.Error("Job failed", 
        zap.String("job", job.Name), 
        zap.Error(err))
    
    // 检查是否需要重试
    if job.Attempts < s.config.Scheduler.RetryAttempts {
        job.Attempts++
        delay := s.calculateRetryDelay(job.Attempts)
        
        s.logger.Info("Scheduling job retry", 
            zap.String("job", job.Name),
            zap.Int("attempt", job.Attempts),
            zap.Duration("delay", delay))
        
        time.AfterFunc(delay, func() {
            s.executeJob(job)
        })
    } else {
        // 达到最大重试次数，发送告警
        s.sendAlert(job, err)
    }
}

func (s *Scheduler) calculateRetryDelay(attempt int) time.Duration {
    baseDelay := time.Duration(s.config.Scheduler.RetryDelay) * time.Second
    maxDelay := 10 * time.Minute
    
    delay := baseDelay * time.Duration(1<<uint(attempt-1))
    if delay > maxDelay {
        return maxDelay
    }
    return delay
}
```

## 日志和审计

```go
type JobAuditLogger struct {
    logger *zap.Logger
}

func (al *JobAuditLogger) LogJobStart(job *Job) {
    al.logger.Info("Job started",
        zap.String("job_name", job.Name),
        zap.String("schedule", job.Schedule),
        zap.Time("scheduled_time", job.NextRun),
    )
}

func (al *JobAuditLogger) LogJobCompletion(job *Job, duration time.Duration, err error) {
    fields := []zap.Field{
        zap.String("job_name", job.Name),
        zap.Duration("execution_time", duration),
        zap.Time("completed_at", time.Now()),
    }
    
    if err != nil {
        fields = append(fields, zap.Error(err))
        al.logger.Error("Job failed", fields...)
    } else {
        al.logger.Info("Job completed successfully", fields...)
    }
}

func (al *JobAuditLogger) LogJobRetry(job *Job, attempt int, delay time.Duration) {
    al.logger.Warn("Job retry scheduled",
        zap.String("job_name", job.Name),
        zap.Int("attempt", attempt),
        zap.Duration("delay", delay),
    )
}
```

## 部署检查清单

- [ ] 配置文件正确设置
- [ ] 数据库连接正常
- [ ] 任务定义完整且有效
- [ ] cron表达式语法正确
- [ ] 日志目录可写
- [ ] 健康检查端点可访问
- [ ] 监控指标正常暴露
- [ ] 告警机制已配置
- [ ] 备份策略已设置
- [ ] 权限配置正确

## 下一步

- [Worker部署指南](../worker/deployment.md)
- [API Server配置](../api-server/deployment.md)
- [基础设施监控](../../infrastructure/monitoring/README.md)