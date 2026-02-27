package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/queue"
	"go.uber.org/zap"
)

// TranscodeService 转码服务
type TranscodeService struct {
	queue      *queue.TaskQueue
	outputDir  string
	workerName string
}

// NewTranscodeService 创建转码服务
func NewTranscodeService(q *queue.TaskQueue, outputDir string) *TranscodeService {
	return &TranscodeService{
		queue:      q,
		outputDir:  outputDir,
		workerName: fmt.Sprintf("transcoder-%d", time.Now().UnixNano()),
	}
}

// StartWorker 启动转码 worker
func (s *TranscodeService) StartWorker(ctx context.Context, concurrency int) error {
	logging.GetLogger().Info("Starting transcode worker",
		zap.String("worker", s.workerName),
		zap.Int("concurrency", concurrency))

	// 启动多个协程并发处理
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			s.processQueue(ctx, workerID)
		}(i)
	}

	return nil
}

// processQueue 处理队列任务
func (s *TranscodeService) processQueue(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			logging.GetLogger().Info("Worker stopping",
				zap.String("worker", s.workerName),
				zap.Int("worker_id", workerID))
			return
		default:
			task, err := s.queue.DequeueTranscodeTask(ctx)
			if err != nil {
				logging.GetLogger().Error("Failed to dequeue task",
					zap.String("worker", s.workerName),
					zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			if task == nil {
				// 队列为空，等待一下
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// 处理任务
			s.processTask(ctx, task, workerID)
		}
	}
}

// processTask 处理单个转码任务
func (s *TranscodeService) processTask(ctx context.Context, task *queue.TranscodeTask, workerID int) {
	logger := logging.GetLogger().With(
		zap.String("task_id", task.TaskID),
		zap.String("video_id", task.VideoID),
		zap.String("worker", s.workerName),
		zap.Int("worker_id", workerID),
	)

	logger.Info("Processing transcode task")

	// 更新任务状态为 processing
	task.Status = queue.TaskStatusProcessing
	now := time.Now()
	task.StartedAt = &now
	s.queue.UpdateTaskStatus(ctx, task)

	// 执行转码
	err := s.executeTranscode(ctx, task)

	if err != nil {
		logger.Error("Transcode failed", zap.Error(err))
		task.Status = queue.TaskStatusFailed
		task.Error = err.Error()
	} else {
		logger.Info("Transcode completed successfully")
		task.Status = queue.TaskStatusCompleted
		task.Progress = 100
	}

	completedAt := time.Now()
	task.CompletedAt = &completedAt
	s.queue.UpdateTaskStatus(ctx, task)

	// 确认任务完成
	if task.MessageID != "" {
		s.queue.AcknowledgeTask(ctx, s.queue.GetQueueName(), task.MessageID)
	}
}

// executeTranscode 执行转码
func (s *TranscodeService) executeTranscode(ctx context.Context, task *queue.TranscodeTask) error {
	// 检查源文件是否存在
	if _, err := os.Stat(task.SourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", task.SourcePath)
	}

	// 生成输出文件名
	outputFilename := fmt.Sprintf("%s_%s.mp4", task.VideoID, task.Quality)
	outputPath := filepath.Join(s.outputDir, outputFilename)

	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 构建 ffmpeg 命令
	var ffmpegArgs []string
	switch task.Quality {
	case "low":
		ffmpegArgs = []string{
			"-i", task.SourcePath,
			"-c:v", "libx264",
			"-b:v", "500k",
			"-maxrate", "500k",
			"-bufsize", "1000k",
			"-vf", "scale=480:-1",
			"-c:a", "aac",
			"-b:a", "64k",
			"-y",
			outputPath,
		}
	case "medium":
		ffmpegArgs = []string{
			"-i", task.SourcePath,
			"-c:v", "libx264",
			"-b:v", "1500k",
			"-maxrate", "1500k",
			"-bufsize", "3000k",
			"-vf", "scale=720:-1",
			"-c:a", "aac",
			"-b:a", "128k",
			"-y",
			outputPath,
		}
	case "high":
		ffmpegArgs = []string{
			"-i", task.SourcePath,
			"-c:v", "libx264",
			"-b:v", "3000k",
			"-maxrate", "3000k",
			"-bufsize", "6000k",
			"-vf", "scale=1280:-1",
			"-c:a", "aac",
			"-b:a", "192k",
			"-y",
			outputPath,
		}
	default:
		ffmpegArgs = []string{
			"-i", task.SourcePath,
			"-c:v", "libx264",
			"-crf", "23",
			"-c:a", "aac",
			"-y",
			outputPath,
		}
	}

	// 检查 ffmpeg 是否存在
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		// 如果没有 ffmpeg，模拟转码（用于测试）
		return s.simulateTranscode(ctx, task)
	}

	// 执行 ffmpeg
	cmd := exec.CommandContext(ctx, ffmpegPath, ffmpegArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}

	// 更新进度
	task.Progress = 100

	return nil
}

// simulateTranscode 模拟转码（用于没有 ffmpeg 的环境）
func (s *TranscodeService) simulateTranscode(ctx context.Context, task *queue.TranscodeTask) error {
	logging.GetLogger().Warn("ffmpeg not found, simulating transcode",
		zap.String("task_id", task.TaskID))

	// 模拟转码进度
	for i := 0; i <= 100; i += 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			task.Progress = i
			s.queue.UpdateTaskStatus(ctx, task)
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// GetQueueStats 获取队列统计
func (s *TranscodeService) GetQueueStats(ctx context.Context) (*queue.QueueStats, error) {
	return s.queue.GetQueueStats(ctx)
}

// GetTaskStatus 获取任务状态
func (s *TranscodeService) GetTaskStatus(ctx context.Context, taskID string) (*queue.TranscodeTask, error) {
	return s.queue.GetTaskStatus(ctx, taskID)
}
