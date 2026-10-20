package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lixiandea/video_server/internal/config"
	"github.com/lixiandea/video_server/internal/models"
	"github.com/lixiandea/video_server/internal/services"
	"github.com/lixiandea/video_server/internal/utils"
	"github.com/lixiandea/video_server/pkg/auth"
	apperrors "github.com/lixiandea/video_server/pkg/errors"
	"github.com/lixiandea/video_server/pkg/logging"
	"github.com/lixiandea/video_server/pkg/metrics"
	"github.com/lixiandea/video_server/pkg/queue"
	"github.com/lixiandea/video_server/pkg/storage"
	"github.com/lixiandea/video_server/pkg/validation"
	"go.uber.org/zap"
)

type VideoHandler struct {
	videoService     *services.VideoService
	likeService      *services.LikeService
	storageService   *storage.StorageService
	transcodeService *services.TranscodeService
	taskQueue        *queue.TaskQueue
	cfg              *config.Config
}

func NewVideoHandler(storageService *storage.StorageService, cfg *config.Config,
	transcodeService *services.TranscodeService, taskQueue *queue.TaskQueue) *VideoHandler {
	return &VideoHandler{
		videoService:     services.NewVideoService(),
		likeService:      services.NewLikeService(),
		storageService:   storageService,
		transcodeService: transcodeService,
		taskQueue:        taskQueue,
		cfg:              cfg,
	}
}

func (h *VideoHandler) UploadVideo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	maxSize := h.cfg.Server.MaxFileSize
	if maxSize <= 0 {
		maxSize = validation.MaxVideoSize
	}

	// 限制请求体大小，避免超大上传耗尽内存或磁盘
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize+(1<<20))

	file, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			utils.Fail(c, http.StatusRequestEntityTooLarge, "File too large")
			return
		}
		utils.Fail(c, http.StatusBadRequest, "Failed to get file from request")
		return
	}

	// Validate file
	if err := validation.ValidateVideoFile(file, maxSize); err != nil {
		if errors.Is(err, validation.ErrFileTooLarge) {
			utils.Fail(c, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		utils.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get video name from form
	videoName := c.PostForm("name")
	if videoName == "" {
		videoName = file.Filename
	}

	if err := validation.ValidateVideoName(videoName); err != nil {
		utils.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// 简介为可选字段，与视频名一起在创建后写入
	description := c.PostForm("description")
	if err := validation.ValidateVideoDescription(description); err != nil {
		utils.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create video record in database
	video, err := h.videoService.CreateVideo(
		userID.(uint),
		videoName,
		file.Size,
		h.cfg.Server.UserVideoQuota,
		h.cfg.Server.UserStorageQuotaBytes,
		h.cfg.Server.UserDailyUploadLimit,
	)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case http.StatusForbidden:
				metrics.IncUploadRejected("quota")
			case http.StatusTooManyRequests:
				metrics.IncUploadRejected("daily_limit")
			}
		}
		utils.FailFrom(c, err, "Failed to create video record")
		return
	}

	// Save file to storage
	src, err := file.Open()
	if err != nil {
		utils.FailFrom(c, err, "Failed to open uploaded file")
		return
	}
	defer src.Close()

	err = h.storageService.SaveVideo(video.UUID, src)
	if err != nil {
		// Clean up the video record if storage fails
		h.videoService.DeleteVideo(video.UUID)
		utils.FailFrom(c, err, "Failed to save video file")
		return
	}

	// Update video with file path and size
	updates := map[string]interface{}{
		"file_path":   h.storageService.SourceKey(video.UUID),
		"size":        file.Size,
		"description": description,
	}
	h.videoService.UpdateVideo(video.UUID, updates)

	// 入队转码任务；Redis 不可用时降级跳过，不影响上传结果
	transcodeStatus := "skipped"
	if h.taskQueue == nil && h.cfg.Server.VideoModerationRequired {
		_ = h.videoService.SetVideoStatus(video.UUID, "failed")
		utils.Fail(c, http.StatusServiceUnavailable, "Transcoding queue is unavailable")
		return
	}
	if h.taskQueue != nil {
		task := &queue.TranscodeTask{
			TaskID:       uuid.New().String(),
			VideoID:      video.UUID,
			SourcePath:   h.storageService.SourceKey(video.UUID),
			TargetFormat: "mp4",
			Quality:      "high", // 可以支持多质量转码
		}
		task.Status = queue.TaskStatusPending

		if err := h.videoService.SaveTranscodeTask(task); err != nil {
			logging.GetLogger().Error("Failed to persist transcode task",
				zap.String("video_id", video.UUID),
				zap.Error(err))
			_ = h.videoService.SetVideoStatus(video.UUID, "failed")
			utils.FailFrom(c, err, "Failed to queue video for transcoding")
			return
		}

		if err := h.taskQueue.EnqueueTranscodeTask(context.Background(), task); err != nil {
			logging.GetLogger().Error("Failed to enqueue transcode task",
				zap.String("video_id", video.UUID),
				zap.Error(err))
			// 持久任务已落库，Scheduler 会从数据库补偿投递。
			transcodeStatus = "queued"
		} else {
			transcodeStatus = "queued"
			logging.GetLogger().Info("Transcode task enqueued",
				zap.String("video_id", video.UUID),
				zap.String("task_id", task.TaskID))
		}
	} else {
		logging.GetLogger().Warn("Transcode queue unavailable, skipping enqueue",
			zap.String("video_id", video.UUID))
	}

	if transcodeStatus == "skipped" {
		// 没有可用的转码任务时原始文件仍可直接播放，避免视频停留在 processing
		finalStatus := "active"
		if h.cfg.Server.VideoModerationRequired {
			finalStatus = "pending_review"
		}
		if err := h.videoService.SetVideoStatus(video.UUID, finalStatus); err != nil {
			logging.GetLogger().Warn("Failed to mark video active after skipping transcode",
				zap.String("video_id", video.UUID),
				zap.Error(err))
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":          "Video uploaded successfully",
		"video_id":         video.UUID,
		"name":             video.Name,
		"transcode_status": transcodeStatus,
	})
}

// resolveVideoPath 返回实际可播放的文件路径：优先使用转码产物（file_path），
// 产物不存在或记录为空时回退到原始上传文件（以 UUID 命名）。
func (h *VideoHandler) resolveVideoPath(video *models.Video) string {
	if h.storageService.IsObjectStorage() {
		if video.FilePath != "" && h.storageService.VideoExists(video.FilePath) {
			return video.FilePath
		}
		return h.storageService.SourceKey(video.UUID)
	}
	if video.FilePath != "" {
		name := filepath.Base(video.FilePath)
		if h.storageService.VideoExists(name) {
			return h.storageService.GetVideoPath(name)
		}
	}
	return h.storageService.GetVideoPath(video.UUID)
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
	videoID := c.Param("video_id")

	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}

	if video.Status != "active" {
		utils.Fail(c, http.StatusNotFound, "Video not found")
		return
	}

	videoPath := h.resolveVideoPath(video)
	c.File(videoPath)
}

func (h *VideoHandler) GetVideoInfo(c *gin.Context) {
	videoID := c.Param("video_id")

	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}

	// processing / failed 也允许查看详情，前端据此展示转码状态；deleted 及未知状态按不存在处理
	switch video.Status {
	case "active", "processing", "failed":
	case "pending_review", "rejected":
		if !h.canViewReviewStatus(c, video.AuthorID) {
			utils.Fail(c, http.StatusNotFound, "Video not found")
			return
		}
	default:
		utils.Fail(c, http.StatusNotFound, "Video not found")
		return
	}

	// 打开详情页即计一次观看：写库并失效缓存；失败不影响内容展示。
	// 转码中的视频尚未可播放，不计入播放量。
	if video.Status == "active" {
		if err := h.videoService.IncrementViewCount(video.UUID); err != nil {
			logging.GetLogger().Warn("Failed to increment view count",
				zap.String("video_id", video.UUID),
				zap.Error(err))
		} else {
			video.ViewCount++
		}
		if userID, ok := c.Get("user_id"); ok {
			if err := h.videoService.RecordVideoView(userID.(uint), video.UUID); err != nil {
				logging.GetLogger().Warn("Failed to record video view",
					zap.String("video_id", video.UUID),
					zap.Uint("user_id", userID.(uint)),
					zap.Error(err))
			}
		}
	}

	// 获取作者信息
	author, err := h.videoService.GetUserByID(video.AuthorID)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get video author")
		return
	}

	// 当前用户的点赞状态；读取失败不影响内容展示
	liked := false
	if userID, ok := c.Get("user_id"); ok {
		if isLiked, err := h.likeService.IsLiked(userID.(uint), video.UUID); err != nil {
			logging.GetLogger().Warn("Failed to load like status",
				zap.String("video_id", video.UUID),
				zap.Error(err))
		} else {
			liked = isLiked
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id":      video.UUID,
		"name":          video.Name,
		"description":   video.Description,
		"author_id":     video.AuthorID,
		"author_name":   author.LoginName,
		"cover_url":     coverURL(video),
		"display_time":  video.DisplayCTime,
		"size":          video.Size,
		"duration":      video.Duration,
		"view_count":    video.ViewCount,
		"like_count":    video.LikeCount,
		"liked":         liked,
		"status":        video.Status,
		"review_note":   video.ReviewNote,
		"reviewed_at":   video.ReviewedAt,
		"hls_available": video.HLSPath != "",
		"created_at":    video.CreatedAt,
	})
}

func (h *VideoHandler) canViewReviewStatus(c *gin.Context, authorID uint) bool {
	value, ok := c.Get("user_id")
	if !ok {
		return false
	}
	userID, ok := value.(uint)
	if !ok {
		return false
	}
	if userID == authorID {
		return true
	}
	role, err := h.videoService.GetUserRole(userID)
	return err == nil && role == "admin"
}

// coverURL 返回封面地址；没有封面时返回空串，由前端回退到占位样式。
// 封面走公开接口：<img> 无法携带 Authorization 头，与播放令牌是同一类限制。
func coverURL(video *models.Video) string {
	if video.CoverPath == "" {
		return ""
	}

	return "/api/v1/videos/" + video.UUID + "/cover"
}

// LikeVideo 点赞（幂等）：重复点赞不会重复计数。
func (h *VideoHandler) LikeVideo(c *gin.Context) {
	h.updateLike(c, true)
}

// UnlikeVideo 取消点赞（幂等）。
func (h *VideoHandler) UnlikeVideo(c *gin.Context) {
	h.updateLike(c, false)
}

func (h *VideoHandler) updateLike(c *gin.Context, like bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	videoID := c.Param("video_id")

	var (
		count int64
		err   error
	)
	if like {
		count, err = h.likeService.Like(userID.(uint), videoID)
	} else {
		count, err = h.likeService.Unlike(userID.(uint), videoID)
	}
	if err != nil {
		utils.FailFrom(c, err, "Failed to update like")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked":      like,
		"like_count": count,
	})
}

// GetVideoCover 提供封面图片，供前端 <img> 直接引用。
func (h *VideoHandler) GetVideoCover(c *gin.Context) {
	videoID := c.Param("video_id")

	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}
	if video.Status != "active" || video.CoverPath == "" {
		utils.Fail(c, http.StatusNotFound, "Cover not found")
		return
	}

	coverName := video.CoverPath
	// ?size=320/720 命中已生成的变体时优先使用，否则回退原图。
	if width, ok := coverSizeWidths[c.Query("size")]; ok {
		variant := filepath.ToSlash(services.CoverVariantPath(video.CoverPath, width))
		if h.storageService.VideoExists(variant) {
			coverName = variant
		}
	}

	if h.storageService.IsObjectStorage() {
		url, err := h.storageService.PresignGet(c.Request.Context(), coverName, time.Hour)
		if err != nil {
			utils.Fail(c, http.StatusNotFound, "Cover not found")
			return
		}
		c.Redirect(http.StatusFound, url)
		return
	}

	coverName = filepath.Base(coverName)
	coverPath := h.storageService.GetVideoPath(coverName)
	if _, err := os.Stat(coverPath); err != nil {
		utils.Fail(c, http.StatusNotFound, "Cover not found")
		return
	}

	// 封面内容由 UUID 决定，可放心让浏览器缓存
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(coverPath)
}

// coverSizeWidths 允许通过 ?size= 请求的封面宽度；未知取值回退原图。
var coverSizeWidths = map[string]int{
	"320": 320,
	"720": 720,
}

// parsePagination 统一解析 page/limit 查询参数，limit 上限 100。
func parsePagination(c *gin.Context) (page, limit, offset int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err = strconv.Atoi(c.DefaultQuery("limit", "12"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 12
	}

	return page, limit, (page - 1) * limit
}

// videoListItem 统一视频列表项的响应结构，列表 / 搜索 / 我的视频 / 相关视频共用，
// 避免各接口字段不一致（前端只依赖 video_id/name/author_name 等字段）。
func videoListItem(video *models.Video, authorName string) map[string]interface{} {
	return map[string]interface{}{
		"video_id":     video.UUID,
		"name":         video.Name,
		"description":  video.Description,
		"author_name":  authorName,
		"cover_url":    coverURL(video),
		"display_time": video.DisplayCTime,
		"size":         video.Size,
		"duration":     video.Duration,
		"view_count":   video.ViewCount,
		"like_count":   video.LikeCount,
		"status":       video.Status,
		"review_note":  video.ReviewNote,
		"reviewed_at":  video.ReviewedAt,
		"created_at":   video.CreatedAt,
	}
}

// buildVideoList 批量补全作者名后构造列表响应，避免逐条查询作者。
func (h *VideoHandler) buildVideoList(videos []*models.Video) ([]map[string]interface{}, error) {
	authorIDs := make([]uint, 0, len(videos))
	for _, v := range videos {
		authorIDs = append(authorIDs, v.AuthorID)
	}

	names, err := h.videoService.GetAuthorNames(authorIDs)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]interface{}, len(videos))
	for i, v := range videos {
		list[i] = videoListItem(v, names[v.AuthorID])
	}
	return list, nil
}

// ListVideos 处理 GET /videos：公开的可播放视频列表，支持 ?q= 按名称搜索。
func (h *VideoHandler) ListVideos(c *gin.Context) {
	page, limit, offset := parsePagination(c)
	query := strings.TrimSpace(c.Query("q"))

	videos, total, err := h.videoService.GetVideos(limit, offset, query)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get videos")
		return
	}

	list, err := h.buildVideoList(videos)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get video authors")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": list,
		"page":   page,
		"limit":  limit,
		"total":  total,
		"query":  query,
	})
}

func optionalUserID(c *gin.Context) *uint {
	value, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	userID, ok := value.(uint)
	if !ok {
		return nil
	}
	return &userID
}

func parseExcludedVideoIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	ids := make([]string, 0)
	for _, id := range strings.Split(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *VideoHandler) buildRecommendationList(items []services.VideoRecommendation) ([]map[string]interface{}, error) {
	videos := make([]*models.Video, len(items))
	for i, item := range items {
		videos[i] = item.Video
	}

	list, err := h.buildVideoList(videos)
	if err != nil {
		return nil, err
	}
	for i, item := range items {
		list[i]["recommendation_reason"] = item.Reason
	}
	return list, nil
}

// GetRecommendations 处理 GET /recommendations：
// 登录用户使用观看行为个性化，匿名用户返回热门/新上架内容。
func (h *VideoHandler) GetRecommendations(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "12"))
	if err != nil || limit < 1 || limit > 50 {
		limit = 12
	}

	items, mode, err := h.videoService.GetRecommendations(
		optionalUserID(c),
		parseExcludedVideoIDs(c.Query("exclude_ids")),
		limit,
	)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get recommendations")
		return
	}

	list, err := h.buildRecommendationList(items)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get recommendation authors")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": list,
		"mode":   mode,
		"limit":  limit,
	})
}

// GetRelatedVideos 处理 GET /videos/:video_id/related：
// 复用推荐排序，并排除当前视频和用户已观看内容。
func (h *VideoHandler) GetRelatedVideos(c *gin.Context) {
	videoID := c.Param("video_id")

	_, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "6"))
	if err != nil || limit < 1 || limit > 50 {
		limit = 6
	}

	items, mode, err := h.videoService.GetRecommendations(optionalUserID(c), []string{videoID}, limit)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get related videos")
		return
	}

	list, err := h.buildRecommendationList(items)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get video authors")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": list,
		"mode":   mode,
	})
}

// GetUserVideos 处理 GET /users/videos：当前用户上传的视频。
func (h *VideoHandler) GetUserVideos(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, limit, offset := parsePagination(c)

	videos, total, err := h.videoService.GetVideosByUserIDWithTotal(userID.(uint), limit, offset)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get videos")
		return
	}

	list, err := h.buildVideoList(videos)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get video authors")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": list,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
}

// ListReviewVideos 返回管理员审核队列。
func (h *VideoHandler) ListReviewVideos(c *gin.Context) {
	status := strings.TrimSpace(c.DefaultQuery("status", "pending_review"))
	switch status {
	case "pending_review", "active", "rejected", "processing", "failed":
	default:
		utils.Fail(c, http.StatusBadRequest, "Invalid review status")
		return
	}
	page, limit, offset := parsePagination(c)
	videos, total, err := h.videoService.GetVideosForReview(status, limit, offset)
	if err != nil {
		utils.FailFrom(c, err, "Failed to list review videos")
		return
	}
	list, err := h.buildVideoList(videos)
	if err != nil {
		utils.FailFrom(c, err, "Failed to get video authors")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"videos": list,
		"page":   page,
		"limit":  limit,
		"total":  total,
		"status": status,
	})
}

// ReviewVideo 批准或拒绝待审视频。
func (h *VideoHandler) ReviewVideo(c *gin.Context) {
	var req struct {
		Decision string `json:"decision" binding:"required"`
		Note     string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, http.StatusBadRequest, "Invalid review request")
		return
	}
	adminID, ok := c.Get("user_id")
	if !ok {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.videoService.ReviewVideo(c.Param("video_id"), adminID.(uint), req.Decision, req.Note); err != nil {
		utils.FailFrom(c, err, "Failed to review video")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Video review completed"})
}

func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	videoID := c.Param("video_id")
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// First get the video to check ownership
	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}

	if video.AuthorID != userID.(uint) {
		utils.Fail(c, http.StatusForbidden, "Not authorized to delete this video")
		return
	}

	// Mark video as deleted in database
	err = h.videoService.DeleteVideo(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Failed to delete video")
		return
	}

	// Add deletion record for cleanup job
	h.videoService.AddDeletionRecord(videoID, "User initiated deletion")

	c.JSON(http.StatusOK, gin.H{"message": "Video marked for deletion"})
}

// CreateStreamToken 处理 GET /videos/:video_id/stream-token：
// 为 <video src> 签发短时效、绑定单个视频的播放令牌。
func (h *VideoHandler) CreateStreamToken(c *gin.Context) {
	videoID := c.Param("video_id")
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}
	if video.Status != "active" {
		utils.Fail(c, http.StatusNotFound, "Video not found")
		return
	}

	ttl := time.Duration(h.cfg.Server.StreamTokenTTL) * time.Second
	if ttl <= 0 {
		ttl = auth.DefaultStreamTokenTTL
	}

	token, err := auth.GenerateStreamToken(userID.(uint), videoID, ttl)
	if err != nil {
		logging.GetLogger().Error("Failed to generate stream token", zap.Error(err))
		utils.FailFrom(c, err, "Failed to generate stream token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": int(ttl.Seconds()),
	})
}

func (h *VideoHandler) GetVideoStream(c *gin.Context) {
	videoID := c.Param("video_id")

	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}

	if video.Status != "active" {
		utils.Fail(c, http.StatusNotFound, "Video not found")
		return
	}

	videoPath := h.resolveVideoPath(video)
	if h.storageService.IsObjectStorage() {
		url, err := h.storageService.PresignGet(c.Request.Context(), videoPath, 15*time.Minute)
		if err != nil {
			utils.FailFrom(c, err, "Failed to create playback URL")
			return
		}
		c.Redirect(http.StatusFound, url)
		return
	}
	c.Header("Content-Type", "video/mp4")
	c.File(videoPath)
}

// GetVideoHLS 提供 HLS 播放列表与分片。
// 播放列表中的相对 URI 会补上 ?token=：浏览器解析相对地址时会丢掉查询参数，
// 否则分片请求无法通过 StreamAuthMiddleware。
func (h *VideoHandler) GetVideoHLS(c *gin.Context) {
	videoID := c.Param("video_id")

	video, err := h.videoService.GetVideoByID(videoID)
	if err != nil {
		utils.FailFrom(c, err, "Video not found")
		return
	}
	if video.Status != "active" || video.HLSPath == "" {
		utils.Fail(c, http.StatusNotFound, "Video not found")
		return
	}

	name := c.Param("file")
	if !validHLSFileName(name) {
		utils.Fail(c, http.StatusNotFound, "File not found")
		return
	}

	objectKey := filepath.ToSlash(filepath.Join(filepath.Dir(video.HLSPath), name))

	if strings.HasSuffix(name, ".m3u8") {
		reader, err := h.storageService.Open(c.Request.Context(), objectKey)
		if err != nil {
			utils.FailFrom(c, err, "Failed to read playlist")
			return
		}
		defer reader.Close()
		content, err := io.ReadAll(reader)
		if err != nil {
			utils.FailFrom(c, err, "Failed to read playlist")
			return
		}
		if token := c.Query("token"); token != "" {
			content = withPlaylistToken(content, token)
		}
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "application/vnd.apple.mpegurl", content)
		return
	}

	if h.storageService.IsObjectStorage() {
		url, err := h.storageService.PresignGet(c.Request.Context(), objectKey, 15*time.Minute)
		if err != nil {
			utils.FailFrom(c, err, "Failed to create segment URL")
			return
		}
		c.Redirect(http.StatusFound, url)
		return
	}

	c.Header("Content-Type", "video/mp2t")
	c.File(objectKey)
}

// validHLSFileName 只放行 playlist 与分片文件名，避免目录穿越。
func validHLSFileName(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return false
	}
	return strings.HasSuffix(name, ".m3u8") || strings.HasSuffix(name, ".ts")
}

// withPlaylistToken 给 playlist 中的相对 URI 追加播放令牌，注释行保持不变。
func withPlaylistToken(content []byte, token string) []byte {
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "?") {
			continue
		}
		lines[i] = trimmed + "?token=" + token
	}
	return []byte(strings.Join(lines, "\n"))
}
