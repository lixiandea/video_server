package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lixiandea/video_server/pkg/database"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/queue"
	"go.uber.org/zap"
)

// TranscodeService 转码服务
type TranscodeService struct {
	queue             *queue.TaskQueue
	outputDir         string
	workerName        string
	videoService      *VideoService
	maxRetries        int
	visibilityTimeout time.Duration
	reclaimInterval   time.Duration
}

// NewTranscodeService 创建转码服务
func NewTranscodeService(q *queue.TaskQueue, outputDir string) *TranscodeService {
	return &TranscodeService{
		queue:             q,
		outputDir:         outputDir,
		workerName:        fmt.Sprintf("transcoder-%d", time.Now().UnixNano()),
		maxRetries:        envInt("TRANSCODE_MAX_RETRIES", 3),
		visibilityTimeout: envDuration("TRANSCODE_VISIBILITY_TIMEOUT", queue.DefaultQueueConfig.VisibilityTimeout),
		reclaimInterval:   envDuration("TRANSCODE_RECLAIM_INTERVAL", time.Minute),
	}
}

// envInt 读取整数环境变量，缺失或非法时返回 fallback。
func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return fallback
}

// envDuration 读取 duration 环境变量（如 30m、10s），缺失或非法时返回 fallback。
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

// StartWorker 启动转码 worker
func (s *TranscodeService) StartWorker(ctx context.Context, concurrency int) error {
	logging.GetLogger().Info("Starting transcode worker",
		zap.String("worker", s.workerName),
		zap.Int("concurrency", concurrency),
		zap.Int("max_retries", s.maxRetries),
		zap.Duration("visibility_timeout", s.visibilityTimeout))

	// 周期性回收崩溃 worker 遗留的 pending 消息
	go s.reclaimLoop(ctx)

	// 启动多个协程并发处理
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			s.processQueue(ctx, workerID)
		}(i)
	}

	return nil
}

// reclaimLoop 定期认领空闲超时的队列消息并重新处理，避免任务永久滞留在 pending。
func (s *TranscodeService) reclaimLoop(ctx context.Context) {
	ticker := time.NewTicker(s.reclaimInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tasks, err := s.queue.ReclaimStaleTasks(ctx, s.visibilityTimeout, 10)
			if err != nil {
				logging.GetLogger().Warn("Failed to reclaim stale transcode tasks", zap.Error(err))
				continue
			}
			for _, task := range tasks {
				logging.GetLogger().Warn("Reclaimed stale transcode task",
					zap.String("task_id", task.TaskID),
					zap.String("video_id", task.VideoID))
				s.processTask(ctx, task, 0)
			}
		}
	}
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

	// 执行转码并落库产物路径
	outputPath, err := s.executeTranscode(ctx, task)
	if err == nil {
		err = s.persistResult(task, outputPath)
	}

	if err != nil {
		logger.Error("Transcode failed", zap.Error(err))
		task.Error = err.Error()

		// 未超过重试上限则重新入队（保留 TaskID），否则标记为永久失败
		if task.Retries < s.maxRetries {
			task.Retries++
			task.Progress = 0
			task.StartedAt = nil
			task.CompletedAt = nil
			if requeueErr := s.queue.EnqueueTranscodeTask(ctx, task); requeueErr != nil {
				logger.Error("Failed to requeue task", zap.Error(requeueErr))
				task.Status = queue.TaskStatusFailed
				completedAt := time.Now()
				task.CompletedAt = &completedAt
				s.queue.UpdateTaskStatus(ctx, task)
				s.markVideoFailed(task.VideoID)
			} else {
				logger.Warn("Transcode task requeued for retry",
					zap.Int("retry", task.Retries),
					zap.Int("max_retries", s.maxRetries))
			}
		} else {
			logger.Error("Transcode task failed permanently", zap.Int("retries", task.Retries))
			task.Status = queue.TaskStatusFailed
			completedAt := time.Now()
			task.CompletedAt = &completedAt
			s.queue.UpdateTaskStatus(ctx, task)
			s.markVideoFailed(task.VideoID)
		}
	} else {
		logger.Info("Transcode completed successfully", zap.String("output_path", outputPath))
		task.Status = queue.TaskStatusCompleted
		task.Progress = 100
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		s.queue.UpdateTaskStatus(ctx, task)
	}

	// 无论成功、重试还是永久失败，原消息都必须确认，
	// 否则它会一直留在 pending 列表并被回收机制重复处理。
	if task.MessageID != "" {
		if err := s.queue.AcknowledgeTask(ctx, s.queue.GetQueueName(), task.MessageID); err != nil {
			logger.Warn("Failed to acknowledge task message", zap.Error(err))
		}
	}
}

// persistResult 把转码产物路径与大小写回视频记录，并失效 Redis 缓存，
// 使 api-server 立即按新路径提供播放；数据库不可用时返回错误（任务会重试）。
func (s *TranscodeService) persistResult(task *queue.TranscodeTask, outputPath string) error {
	if database.DB == nil {
		return fmt.Errorf("database not initialized, cannot persist transcode result")
	}
	if s.videoService == nil {
		s.videoService = NewVideoService()
	}

	updates := map[string]interface{}{"file_path": outputPath}
	if info, err := os.Stat(outputPath); err == nil {
		updates["size"] = info.Size()
	}
	s.extractMetadata(outputPath, updates)

	if err := s.videoService.UpdateVideo(task.VideoID, updates); err != nil {
		return fmt.Errorf("failed to persist transcode result: %w", err)
	}
	if err := s.videoService.SetVideoStatus(task.VideoID, "active"); err != nil {
		return fmt.Errorf("failed to mark video active: %w", err)
	}

	return nil
}

// markVideoFailed 在重试耗尽或无法重新入队时把视频标记为转码失败。
func (s *TranscodeService) markVideoFailed(videoID string) {
	if database.DB == nil {
		return
	}
	if s.videoService == nil {
		s.videoService = NewVideoService()
	}
	if err := s.videoService.SetVideoStatus(videoID, "failed"); err != nil {
		logging.GetLogger().Error("Failed to mark video as failed",
			zap.String("video_id", videoID),
			zap.Error(err))
	}
}

// extractMetadata 在转码成功后补充时长与封面（P1-11b）。
// 两项都属于「锦上添花」：失败只告警，不影响转码任务的成功状态。
func (s *TranscodeService) extractMetadata(outputPath string, updates map[string]interface{}) {
	logger := logging.GetLogger()

	if duration, err := probeDuration(outputPath); err != nil {
		logger.Warn("Failed to probe video duration",
			zap.String("path", outputPath),
			zap.Error(err))
	} else if duration > 0 {
		updates["duration"] = duration
	}

	coverPath := coverPathFor(outputPath)
	if err := extractCover(outputPath, coverPath); err != nil {
		logger.Warn("Failed to extract video cover",
			zap.String("path", outputPath),
			zap.Error(err))
		return
	}
	updates["cover_path"] = coverPath
}

// coverPathFor 封面与转码产物放在同一目录，命名为 <产物名>_cover.jpg。
func coverPathFor(outputPath string) string {
	return strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_cover.jpg"
}

// probeDuration 用 ffprobe 读取时长（秒）。ffprobe 缺失或输出异常时返回错误。
func probeDuration(path string) (float64, error) {
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe error: %w", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("unexpected ffprobe output %q: %w", string(out), err)
	}

	return duration, nil
}

// extractCover 抽取一帧作为封面。先尝试第 1 秒，短视频取不到帧时回退到首帧。
func extractCover(sourcePath, coverPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var lastOutput []byte
	for _, seek := range []string{"1", "0"} {
		out, err := exec.CommandContext(ctx, ffmpegPath,
			"-y", "-ss", seek, "-i", sourcePath,
			"-vframes", "1", "-q:v", "2", coverPath).CombinedOutput()
		lastOutput = out
		if err != nil {
			continue
		}
		if info, statErr := os.Stat(coverPath); statErr == nil && info.Size() > 0 {
			return nil
		}
	}

	return fmt.Errorf("ffmpeg could not extract a cover frame: %s", string(lastOutput))
}

// executeTranscode 执行转码，返回产物路径
func (s *TranscodeService) executeTranscode(ctx context.Context, task *queue.TranscodeTask) (string, error) {
	// 检查源文件是否存在
	if _, err := os.Stat(task.SourcePath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file not found: %s", task.SourcePath)
	}

	// 生成输出文件名
	outputFilename := fmt.Sprintf("%s_%s.mp4", task.VideoID, task.Quality)
	outputPath := filepath.Join(s.outputDir, outputFilename)

	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
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
		// 缺少 ffmpeg 时必须失败：此前“模拟转码”会把任务标记为完成但没有产物，
		// 导致上层以为转码成功、播放却找不到文件。
		return "", fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// 执行 ffmpeg
	cmd := exec.CommandContext(ctx, ffmpegPath, ffmpegArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}

	// 更新进度
	task.Progress = 100

	return outputPath, nil
}

// GetQueueStats 获取队列统计
func (s *TranscodeService) GetQueueStats(ctx context.Context) (*queue.QueueStats, error) {
	return s.queue.GetQueueStats(ctx)
}

// GetTaskStatus 获取任务状态
func (s *TranscodeService) GetTaskStatus(ctx context.Context, taskID string) (*queue.TranscodeTask, error) {
	return s.queue.GetTaskStatus(ctx, taskID)
}
