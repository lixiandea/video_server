package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/config"
    "github.com/lixiandea/video_server/internal/services"
    "github.com/lixiandea/video_server/pkg/storage"
    "github.com/lixiandea/video_server/pkg/validation"
)

type VideoHandler struct {
    videoService  *services.VideoService
    storageService *storage.StorageService
    cfg           *config.Config
}

func NewVideoHandler(storageService *storage.StorageService, cfg *config.Config) *VideoHandler {
    return &VideoHandler{
        videoService:  services.NewVideoService(),
        storageService: storageService,
        cfg:           cfg,
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

    c.JSON(http.StatusCreated, gin.H{
        "message": "Video uploaded successfully",
        "video_id": video.UUID,
        "name":    video.Name,
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

    c.JSON(http.StatusOK, gin.H{
        "video_id":     video.UUID,
        "name":        video.Name,
        "author_id":   video.AuthorID,
        "author_name": video.Author.LoginName,
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

    videos, err := h.videoService.GetVideosByUserID(userID.(uint), limit, offset)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get videos"})
        return
    }

    // Convert to response format
    videoList := make([]map[string]interface{}, len(videos))
    for i, video := range videos {
        videoList[i] = map[string]interface{}{
            "video_id":     video.UUID,
            "name":        video.Name,
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
        "total":  len(videoList), // In a real app, you'd want to return actual total count
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