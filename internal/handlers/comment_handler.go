package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/services"
)

type CommentHandler struct {
    commentService *services.CommentService
    videoService   *services.VideoService
}

func NewCommentHandler() *CommentHandler {
    return &CommentHandler{
        commentService: services.NewCommentService(),
        videoService:   services.NewVideoService(),
    }
}

func (h *CommentHandler) AddComment(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    videoIDStr := c.Param("video_id")
    videoID, err := strconv.ParseUint(videoIDStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
        return
    }

    var req struct {
        Content string `json:"content" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    // Validate content
    if len(req.Content) == 0 || len(req.Content) > 1000 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Content must be between 1 and 1000 characters"})
        return
    }

    // Check if video exists and is active
    video, err := h.videoService.GetVideoByID(videoIDStr)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }
    if video.Status != "active" {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    comment, err := h.commentService.AddComment(uint(videoID), userID.(uint), req.Content)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "Comment added successfully",
        "comment_id": comment.UUID,
        "content": comment.Content,
        "created_at": comment.CreatedAt,
    })
}

func (h *CommentHandler) GetComments(c *gin.Context) {
    videoIDStr := c.Param("video_id")
    videoID, err := strconv.ParseUint(videoIDStr, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
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

    // Check if video exists and is active
    video, err := h.videoService.GetVideoByID(videoIDStr)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }
    if video.Status != "active" {
        c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
        return
    }

    comments, err := h.commentService.GetCommentsByVideoID(uint(videoID), limit, offset)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comments"})
        return
    }

    // Convert to response format
    commentList := make([]map[string]interface{}, len(comments))
    for i, comment := range comments {
        commentList[i] = map[string]interface{}{
            "comment_id": comment.UUID,
            "author_id":  comment.AuthorID,
            "author_name": comment.Author.LoginName,
            "content":    comment.Content,
            "created_at": comment.CreatedAt,
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "comments": commentList,
        "page":     page,
        "limit":    limit,
        "total":    len(commentList), // In a real app, you'd want to return actual total count
    })
}

func (h *CommentHandler) GetComment(c *gin.Context) {
    commentID := c.Param("comment_id")

    comment, err := h.commentService.GetCommentByID(commentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "comment_id": comment.UUID,
        "video_id":   comment.VideoID,
        "author_id":  comment.AuthorID,
        "author_name": comment.Author.LoginName,
        "content":    comment.Content,
        "created_at": comment.CreatedAt,
    })
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    commentID := c.Param("comment_id")

    var req struct {
        Content string `json:"content" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    // Validate content
    if len(req.Content) == 0 || len(req.Content) > 1000 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Content must be between 1 and 1000 characters"})
        return
    }

    // Get the comment to check ownership
    comment, err := h.commentService.GetCommentByID(commentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
        return
    }

    if comment.AuthorID != userID.(uint) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this comment"})
        return
    }

    updates := map[string]interface{}{
        "content": req.Content,
    }

    err = h.commentService.UpdateComment(commentID, updates)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Comment updated successfully"})
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    commentID := c.Param("comment_id")

    // Get the comment to check ownership
    comment, err := h.commentService.GetCommentByID(commentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
        return
    }

    if comment.AuthorID != userID.(uint) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this comment"})
        return
    }

    err = h.commentService.DeleteComment(commentID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}