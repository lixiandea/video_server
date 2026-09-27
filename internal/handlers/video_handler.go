package handlers

import (
    "context"
    "errors"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/config"
    "github.com/lixiandea/video_server/pkg/auth"
    "github.com/lixiandea/video_server/internal/models"
    "github.com/lixiandea/video_server/internal/services"
    "github.com/lixiandea/video_server/internal/utils"
    "github.com/lixiandea/video_server/pkg/logging"
    "github.com/lixiandea/video_server/pkg/queue"
    "github.com/lixiandea/video_server/pkg/storage"
    "github.com/lixiandea/video_server/pkg/validation"
    "go.uber.org/zap"
)

type VideoHandler struct {
    videoService   *services.VideoService
    storageService *storage.StorageService
    transcodeService *services.TranscodeService
    taskQueue      *queue.TaskQueue
    cfg            *config.Config
}

func NewVideoHandler(storageService *storage.StorageService, cfg *config.Config, 
    transcodeService *services.TranscodeService, taskQueue *queue.TaskQueue) *VideoHandler {
    return &VideoHandler{
        videoService:     services.NewVideoService(),
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
    video, err := h.videoService.CreateVideo(userID.(uint), videoName)
    if err != nil {
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
        "file_path":   h.storageService.GetVideoPath(video.UUID),
        "size":        file.Size,
        "description": description,
    }
    h.videoService.UpdateVideo(video.UUID, updates)

    // 入队转码任务；Redis 不可用时降级跳过，不影响上传结果
    transcodeStatus := "skipped"
    if h.taskQueue != nil {
        task := &queue.TranscodeTask{
            VideoID:      video.UUID,
            SourcePath:   h.storageService.GetVideoPath(video.UUID),
            TargetFormat: "mp4",
            Quality:      "high", // 可以支持多质量转码
        }

        if err := h.taskQueue.EnqueueTranscodeTask(context.Background(), task); err != nil {
            logging.GetLogger().Error("Failed to enqueue transcode task",
                zap.String("video_id", video.UUID),
                zap.Error(err))
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

    c.JSON(http.StatusCreated, gin.H{
        "message": "Video uploaded successfully",
        "video_id": video.UUID,
        "name":    video.Name,
        "transcode_status": transcodeStatus,
    })
}

// resolveVideoPath 返回实际可播放的文件路径：优先使用转码产物（file_path），
// 产物不存在或记录为空时回退到原始上传文件（以 UUID 命名）。
func (h *VideoHandler) resolveVideoPath(video *models.Video) string {
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

    if video.Status != "active" {
        utils.Fail(c, http.StatusNotFound, "Video not found")
        return
    }

    // 打开详情页即计一次观看：写库并失效缓存；失败不影响内容展示
    if err := h.videoService.IncrementViewCount(video.UUID); err != nil {
        logging.GetLogger().Warn("Failed to increment view count",
            zap.String("video_id", video.UUID),
            zap.Error(err))
    } else {
        video.ViewCount++
    }

    // 获取作者信息
    author, err := h.videoService.GetUserByID(video.AuthorID)
    if err != nil {
        utils.FailFrom(c, err, "Failed to get video author")
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "video_id":     video.UUID,
        "name":         video.Name,
        "description":  video.Description,
        "author_id":    video.AuthorID,
        "author_name":  author.LoginName,
        "cover_url":    coverURL(video),
        "display_time": video.DisplayCTime,
        "size":         video.Size,
        "duration":     video.Duration,
        "view_count":   video.ViewCount,
        "status":       video.Status,
        "created_at":   video.CreatedAt,
    })
}

// coverURL 返回封面地址；没有封面时返回空串，由前端回退到占位样式。
// 封面走公开接口：<img> 无法携带 Authorization 头，与播放令牌是同一类限制。
func coverURL(video *models.Video) string {
    if video.CoverPath == "" {
        return ""
    }

    return "/api/v1/videos/" + video.UUID + "/cover"
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

    // 与播放路径一致：库中可能是别处写入的绝对路径，这里只取文件名重新拼接。
    coverPath := h.storageService.GetVideoPath(filepath.Base(video.CoverPath))
    if _, err := os.Stat(coverPath); err != nil {
        utils.Fail(c, http.StatusNotFound, "Cover not found")
        return
    }

    // 封面内容由 UUID 决定，可放心让浏览器缓存
    c.Header("Cache-Control", "public, max-age=3600")
    c.File(coverPath)
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
        "status":       video.Status,
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

// GetRelatedVideos 处理 GET /videos/:video_id/related：同作者的其他视频。
func (h *VideoHandler) GetRelatedVideos(c *gin.Context) {
    videoID := c.Param("video_id")

    video, err := h.videoService.GetVideoByID(videoID)
    if err != nil {
        utils.FailFrom(c, err, "Video not found")
        return
    }

    limit, err := strconv.Atoi(c.DefaultQuery("limit", "6"))
    if err != nil || limit < 1 || limit > 50 {
        limit = 6
    }

    videos, err := h.videoService.GetRelatedVideos(videoID, video.AuthorID, limit)
    if err != nil {
        utils.FailFrom(c, err, "Failed to get related videos")
        return
    }

    list, err := h.buildVideoList(videos)
    if err != nil {
        utils.FailFrom(c, err, "Failed to get video authors")
        return
    }

    c.JSON(http.StatusOK, gin.H{"videos": list})
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
    c.Header("Content-Type", "video/mp4")
    c.File(videoPath)
}
