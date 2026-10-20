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
	"github.com/lixiandea/video_server/pkg/storage"
	"go.uber.org/zap"
)

// TranscodeService 转码服务
type TranscodeService struct {
	queue              *queue.TaskQueue
	outputDir          string
	workerName         string
	videoService       *VideoService
	maxRetries         int
	visibilityTimeout  time.Duration
	reclaimInterval    time.Duration
	moderationRequired bool
	storage            *storage.StorageService
}

// NewTranscodeService 创建转码服务
func NewTranscodeService(q *queue.TaskQueue, storageService *storage.StorageService, moderationRequired bool) *TranscodeService {
	return &TranscodeService{
		queue:              q,
		outputDir:          storageService.OutputDir(),
		workerName:         fmt.Sprintf("transcoder-%d", time.Now().UnixNano()),
		maxRetries:         envInt("TRANSCODE_MAX_RETRIES", 3),
		visibilityTimeout:  envDuration("TRANSCODE_VISIBILITY_TIMEOUT", queue.DefaultQueueConfig.VisibilityTimeout),
		reclaimInterval:    envDuration("TRANSCODE_RECLAIM_INTERVAL", time.Minute),
		moderationRequired: moderationRequired,
		storage:            storageService,
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
	s.recoverPersistentTasks(ctx)

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
			s.recoverPersistentTasks(ctx)
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

// recoverPersistentTasks 从数据库恢复 Redis 消息丢失或 Worker 启动前积压的任务。
func (s *TranscodeService) recoverPersistentTasks(ctx context.Context) {
	if s.videoService == nil {
		s.videoService = NewVideoService()
	}
	tasks, err := s.videoService.LoadRecoverableTranscodeTasks(time.Now().Add(-s.visibilityTimeout), 20)
	if err != nil {
		logging.GetLogger().Warn("Failed to load recoverable transcode tasks", zap.Error(err))
		return
	}
	for _, task := range tasks {
		if err := s.queue.EnqueueTranscodeTask(ctx, task); err != nil {
			logging.GetLogger().Warn("Failed to recover transcode task",
				zap.String("task_id", task.TaskID),
				zap.Error(err))
			continue
		}
		if err := s.videoService.SaveTranscodeTask(task); err != nil {
			logging.GetLogger().Warn("Failed to update recovered task", zap.Error(err))
		}
		logging.GetLogger().Info("Recovered transcode task from database",
			zap.String("task_id", task.TaskID),
			zap.String("video_id", task.VideoID))
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
	s.persistTaskState(task)

	// 执行转码并落库产物路径
	outputPath, hlsPath, err := s.executeTranscode(ctx, task)
	if err == nil {
		err = s.persistResult(task, outputPath, hlsPath)
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
				s.persistTaskState(task)
				s.markVideoFailed(task.VideoID)
			} else {
				s.persistTaskState(task)
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
			s.persistTaskState(task)
			s.markVideoFailed(task.VideoID)
		}
	} else {
		logger.Info("Transcode completed successfully", zap.String("output_path", outputPath))
		task.Status = queue.TaskStatusCompleted
		task.Progress = 100
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		s.queue.UpdateTaskStatus(ctx, task)
		s.persistTaskState(task)
	}

	// 无论成功、重试还是永久失败，原消息都必须确认，
	// 否则它会一直留在 pending 列表并被回收机制重复处理。
	if task.MessageID != "" {
		if err := s.queue.AcknowledgeTask(ctx, s.queue.GetQueueName(), task.MessageID); err != nil {
			logger.Warn("Failed to acknowledge task message", zap.Error(err))
		}
	}
}

func (s *TranscodeService) persistTaskState(task *queue.TranscodeTask) {
	if database.DB == nil {
		return
	}
	if s.videoService == nil {
		s.videoService = NewVideoService()
	}
	if err := s.videoService.SaveTranscodeTask(task); err != nil {
		logging.GetLogger().Warn("Failed to persist task state",
			zap.String("task_id", task.TaskID),
			zap.Error(err))
	}
}

// persistResult 把转码产物路径与大小写回视频记录，并失效 Redis 缓存，
// 使 api-server 立即按新路径提供播放；数据库不可用时返回错误（任务会重试）。
func (s *TranscodeService) persistResult(task *queue.TranscodeTask, outputPath, hlsPath string) error {
	if database.DB == nil {
		return fmt.Errorf("database not initialized, cannot persist transcode result")
	}
	if s.videoService == nil {
		s.videoService = NewVideoService()
	}

	updates := map[string]interface{}{}
	if info, err := os.Stat(outputPath); err == nil {
		updates["size"] = info.Size()
	}
	s.extractMetadata(outputPath, updates)

	storedOutputPath := outputPath
	storedHLSPath := hlsPath
	if s.storage.IsObjectStorage() {
		outputKey := "videos/" + task.VideoID + "/" + filepath.Base(outputPath)
		if err := s.storage.UploadFile(context.Background(), outputKey, outputPath, "video/mp4"); err != nil {
			return err
		}
		storedOutputPath = outputKey

		hlsPrefix := "videos/" + task.VideoID + "/hls"
		if err := s.storage.UploadDir(context.Background(), hlsPrefix, filepath.Dir(hlsPath)); err != nil {
			return err
		}
		storedHLSPath = hlsPrefix + "/index.m3u8"

		if coverPath, ok := updates["cover_path"].(string); ok {
			coverKey := "videos/" + task.VideoID + "/" + filepath.Base(coverPath)
			if err := s.storage.UploadFile(context.Background(), coverKey, coverPath, "image/jpeg"); err != nil {
				return err
			}
			for _, width := range coverVariantWidths {
				variant := CoverVariantPath(coverPath, width)
				if _, err := os.Stat(variant); err != nil {
					continue
				}
				variantKey := "videos/" + task.VideoID + "/" + filepath.Base(variant)
				if err := s.storage.UploadFile(context.Background(), variantKey, variant, "image/jpeg"); err != nil {
					return err
				}
			}
			updates["cover_path"] = coverKey
		}
		defer os.RemoveAll(filepath.Dir(hlsPath))
		defer os.Remove(outputPath)
	}
	updates["file_path"] = storedOutputPath
	if storedHLSPath != "" {
		updates["hls_path"] = storedHLSPath
	}

	if err := s.videoService.UpdateVideo(task.VideoID, updates); err != nil {
		return fmt.Errorf("failed to persist transcode result: %w", err)
	}
	finalStatus := "active"
	if s.moderationRequired {
		finalStatus = "pending_review"
	}
	if err := s.videoService.SetVideoStatus(task.VideoID, finalStatus); err != nil {
		return fmt.Errorf("failed to mark video %s: %w", finalStatus, err)
	}
	if s.storage.IsObjectStorage() {
		backupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err := s.storage.CopyVideoToBackup(backupCtx, task.VideoID); err != nil {
			logging.GetLogger().Warn("Immediate media backup failed; scheduler will retry",
				zap.String("video_id", task.VideoID),
				zap.Error(err))
		} else if err := s.videoService.MarkAssetsReplicated(task.VideoID); err != nil {
			logging.GetLogger().Warn("Failed to record media replication", zap.Error(err))
		}
		cancel()
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
	for _, width := range coverVariantWidths {
		variant := CoverVariantPath(coverPath, width)
		if err := scaleCover(coverPath, variant, width); err != nil {
			logger.Warn("Failed to generate cover variant",
				zap.String("path", variant),
				zap.Int("width", width),
				zap.Error(err))
		}
	}
}

// coverPathFor 封面与转码产物放在同一目录，命名为 <产物名>_cover.jpg。
func coverPathFor(outputPath string) string {
	return strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "_cover.jpg"
}

// coverVariantWidths 是多分辨率封面的宽度档位（与接口 ?size= 的取值一致）。
var coverVariantWidths = []int{320, 720}

// CoverVariantPath 返回封面指定宽度的变体路径（命名约定与转码侧一致）。
func CoverVariantPath(coverPath string, width int) string {
	return fmt.Sprintf("%s_%dw.jpg", strings.TrimSuffix(coverPath, filepath.Ext(coverPath)), width)
}

// scaleCover 把已有封面缩放成指定宽度的 JPEG。
func scaleCover(src, dst string, width int) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, ffmpegPath,
		"-y", "-i", src,
		"-vf", fmt.Sprintf("scale=%d:-2", width),
		"-q:v", "3", dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg scale failed: %w, output: %s", err, string(out))
	}
	return nil
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

// executeTranscode 执行 MP4 转码并打包多码率 HLS，返回 (MP4 路径, master playlist 路径)。
func (s *TranscodeService) executeTranscode(ctx context.Context, task *queue.TranscodeTask) (string, string, error) {
	sourcePath, err := s.storage.Materialize(ctx, task.SourcePath)
	if err != nil {
		return "", "", fmt.Errorf("source file not found: %s: %w", task.SourcePath, err)
	}
	if s.storage.IsObjectStorage() {
		defer os.Remove(sourcePath)
	}
	if err := scanForMalware(ctx, sourcePath); err != nil {
		return "", "", err
	}

	// 生成输出文件名
	outputFilename := fmt.Sprintf("%s_%s.mp4", task.VideoID, task.Quality)
	outputPath := filepath.Join(s.outputDir, outputFilename)

	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 构建 ffmpeg 命令
	var ffmpegArgs []string
	switch task.Quality {
	case "low":
		ffmpegArgs = []string{
			"-i", sourcePath,
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
			"-i", sourcePath,
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
			"-i", sourcePath,
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
			"-i", sourcePath,
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
		return "", "", fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// 执行 ffmpeg
	cmd := exec.CommandContext(ctx, ffmpegPath, ffmpegArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}

	// 更新进度
	task.Progress = 100

	hlsPath, err := s.packHLS(ctx, sourcePath, task.VideoID)
	if err != nil {
		return "", "", fmt.Errorf("failed to package HLS: %w", err)
	}

	return outputPath, hlsPath, nil
}

func scanForMalware(ctx context.Context, path string) error {
	if os.Getenv("MALWARE_SCAN_ENABLED") != "true" {
		return nil
	}
	scanner, err := exec.LookPath("clamdscan")
	if err != nil {
		return fmt.Errorf("malware scanning enabled but clamdscan is unavailable: %w", err)
	}
	args := []string{"--no-summary"}
	if host := os.Getenv("CLAMAV_HOST"); host != "" {
		args = append(args, "--host="+host)
	}
	args = append(args, path)
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	output, err := exec.CommandContext(scanCtx, scanner, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("malware scan failed: %w, output: %s", err, string(output))
	}
	return nil
}

// hlsRendition 描述一路 HLS 码率。
type hlsRendition struct {
	name      string
	height    int
	bitrate   string
	bufsize   string
	audioRate string
	bandwidth int
}

var hlsRenditions = []hlsRendition{
	{name: "480p", height: 480, bitrate: "800k", bufsize: "1600k", audioRate: "64k", bandwidth: 900000},
	{name: "720p", height: 720, bitrate: "1500k", bufsize: "3000k", audioRate: "96k", bandwidth: 1650000},
}

// packHLS 把源文件切成多码率 HLS（VOD），生成 master playlist 并返回其路径。
// ffmpeg 的 hls 封装会把 -hls_segment_filename 原样写进 playlist，
// 因此把工作目录切到输出目录、只传相对文件名，保证分片 URI 是相对的。
func (s *TranscodeService) packHLS(ctx context.Context, sourcePath, videoID string) (string, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve source path: %w", err)
	}

	dir := filepath.Join(s.outputDir, videoID+"_hls")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create HLS directory: %w", err)
	}

	var master strings.Builder
	master.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n")
	for _, rendition := range hlsRenditions {
		playlist := rendition.name + ".m3u8"
		args := []string{
			"-i", absSource,
			"-c:v", "libx264",
			"-b:v", rendition.bitrate,
			"-maxrate", rendition.bitrate,
			"-bufsize", rendition.bufsize,
			"-vf", fmt.Sprintf("scale=-2:%d", rendition.height),
			"-c:a", "aac",
			"-b:a", rendition.audioRate,
			"-hls_time", "4",
			"-hls_playlist_type", "vod",
			"-hls_segment_filename", rendition.name + "_%03d.ts",
			"-y", playlist,
		}
		cmd := exec.CommandContext(ctx, ffmpegPath, args...)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("ffmpeg HLS %s failed: %w, output: %s", rendition.name, err, string(output))
		}
		master.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d\n%s\n", rendition.bandwidth, playlist))
	}

	masterPath := filepath.Join(dir, "index.m3u8")
	if err := os.WriteFile(masterPath, []byte(master.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write HLS master playlist: %w", err)
	}

	return masterPath, nil
}

// GetQueueStats 获取队列统计
func (s *TranscodeService) GetQueueStats(ctx context.Context) (*queue.QueueStats, error) {
	return s.queue.GetQueueStats(ctx)
}

// GetTaskStatus 获取任务状态
func (s *TranscodeService) GetTaskStatus(ctx context.Context, taskID string) (*queue.TranscodeTask, error) {
	return s.queue.GetTaskStatus(ctx, taskID)
}
