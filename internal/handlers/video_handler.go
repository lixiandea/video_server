package handlers

import (
    "context"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/config"
    "github.com/lixiandea/video_server/internal/services"
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
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request"})
        return
    }

    // Validate file
    if err := validation.ValidateVideoFile(file); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Get video name from form
    videoName := c.PostForm("name")
    if videoName == "" {
        videoName = file.Filename
    }

    if err := validation.ValidateVideoName(videoName); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Create video record in database
    video, err := h.videoService.CreateVideo(userID.(uint), videoName)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video record"})
        return
    }

    // Save file to storage
    src, err := file.Open()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
        return
    }
    defer src.Close()

    err = h.storageService.SaveVideo(video.UUID, src)
    if err != nil {
        // Clean up the video record if storage fails
        h.videoService.DeleteVideo(video.UUID)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
        return
    }

    // Update video with file path and size
    updates := map[string]interface{}{
        "file_path": h.storageService.GetVideoPath(video.UUID),
        "size":      file.Size,
    }
    h.videoService.UpdateVideo(video.UUID, updates)

    // 添加转码任务到队列（异步处理，不影响上传响应）
    go func() {
        task := &queue.TranscodeTask{
            VideoID:      video.UUID,
            SourcePath:   h.storageService.GetVideoPath(video.UUID),
            TargetFormat: "mp4",
            Quality:      "high", // 可以支持多质量转码
        }
        
        ctx := context.Background()
        if err := h.taskQueue.EnqueueTranscodeTask(ctx, task); err != nil {
            logging.GetLogger().Error("Failed to enqueue transcode task",
                zap.String("video_id", video.UUID),
                zap.Error(err))
        } else {
            logging.GetLogger().Info("Transcode task enqueued",
                zap.String("video_id", video.UUID),
                zap.String("task_id", task.TaskID))
        }
    }()

    c.JSON(http.StatusCreated, gin.H{
        "message": "Video uploaded successfully",
        "video_id": video.UUID,
        "name":    video.Name,
        "transcode_status": "queued",
    })
}

func (h *VideoHandler) GetVideo(c *gin.Context) {
    videoID := c.Param("video_id")

    video, err := h.videoService.GetVideoByID(videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    if video.Status != "active" {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    videoPath := h.storageService.GetVideoPath(video.UUID)
    c.File(videoPath)
}

func (h *VideoHandler) GetVideoInfo(c *gin.Context) {
    videoID := c.Param("video_id")

    video, err := h.videoService.GetVideoByID(videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    if video.Status != "active" {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    // 获取作者信息
    author, err := h.videoService.GetUserByID(video.AuthorID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get video author"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "video_id":     video.UUID,
        "name":        video.Name,
        "author_id":   video.AuthorID,
        "author_name": author.LoginName,
        "display_time": video.DisplayCTime,
        "size":        video.Size,
        "status":      video.Status,
        "created_at":  video.CreatedAt,
    })
}

func (h *VideoHandler) GetUserVideos(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // Get pagination params
    pageStr := c.DefaultQuery("page", "1")
    limitStr := c.DefaultQuery("limit", "10")

    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        page = 1
    }

    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit < 1 || limit > 100 {
        limit = 10
    }

    offset := (page - 1) * limit

    videos, total, err := h.videoService.GetVideosByUserIDWithTotal(userID.(uint), limit, offset)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get videos"})
        return
    }

    // Convert to response format
    videoList := make([]map[string]interface{}, len(videos))
    for i, video := range videos {
        // 获取作者信息
        author, err := h.videoService.GetUserByID(video.AuthorID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get video author"})
            return
        }

        videoList[i] = map[string]interface{}{
            "video_id":     video.UUID,
            "name":        video.Name,
            "author_name": author.LoginName,
            "display_time": video.DisplayCTime,
            "size":        video.Size,
            "status":      video.Status,
            "created_at":  video.CreatedAt,
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "videos": videoList,
        "page":   page,
        "limit":  limit,
        "total":  total,
    })
}

func (h *VideoHandler) DeleteVideo(c *gin.Context) {
    videoID := c.Param("video_id")
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // First get the video to check ownership
    video, err := h.videoService.GetVideoByID(videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    if video.AuthorID != userID.(uint) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this video"})
        return
    }

    // Mark video as deleted in database
    err = h.videoService.DeleteVideo(videoID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video"})
        return
    }

    // Add deletion record for cleanup job
    h.videoService.AddDeletionRecord(videoID, "User initiated deletion")

    c.JSON(http.StatusOK, gin.H{"message": "Video marked for deletion"})
}

func (h *VideoHandler) GetVideoStream(c *gin.Context) {
    videoID := c.Param("video_id")

    video, err := h.videoService.GetVideoByID(videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    if video.Status != "active" {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    videoPath := h.storageService.GetVideoPath(video.UUID)
    c.Header("Content-Type", "video/mp4")
    c.File(videoPath)
}