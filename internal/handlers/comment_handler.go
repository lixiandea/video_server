package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/lixiandea/video_server/internal/services"
    "github.com/lixiandea/video_server/internal/utils"
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
        utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
        return
    }

    videoIDStr := c.Param("video_id")
    if videoIDStr == "" {
        utils.Fail(c, http.StatusBadRequest, "Video ID is required")
        return
    }

    var req struct {
        Content string `json:"content" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        utils.Fail(c, http.StatusBadRequest, "Invalid request format")
        return
    }

    // Validate content
    if len(req.Content) == 0 || len(req.Content) > 1000 {
        utils.Fail(c, http.StatusBadRequest, "Content must be between 1 and 1000 characters")
        return
    }

    // Check if video exists and is active
    video, err := h.videoService.GetVideoByID(videoIDStr)
    if err != nil {
        utils.FailFrom(c, err, "Video not found")
        return
    }
    if video.Status != "active" {
        utils.Fail(c, http.StatusNotFound, "Video not found")
        return
    }

    comment, err := h.commentService.AddComment(videoIDStr, userID.(uint), req.Content)
    if err != nil {
        utils.FailFrom(c, err, "Failed to add comment")
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
    if videoIDStr == "" {
        utils.Fail(c, http.StatusBadRequest, "Video ID is required")
        return
    }

    page, limit, offset := parsePagination(c)

    // Check if video exists and is active
    video, err := h.videoService.GetVideoByID(videoIDStr)
    if err != nil {
        utils.FailFrom(c, err, "Video not found")
        return
    }
    if video.Status != "active" {
        utils.Fail(c, http.StatusNotFound, "Video not found")
        return
    }

    comments, total, err := h.commentService.GetCommentsByVideoIDWithTotal(videoIDStr, limit, offset)
    if err != nil {
        utils.FailFrom(c, err, "Failed to get comments")
        return
    }

    // 批量取作者名，避免每条评论一次查询（N+1）
    authorIDs := make([]uint, 0, len(comments))
    for _, comment := range comments {
        authorIDs = append(authorIDs, comment.AuthorID)
    }
    authorNames, err := h.videoService.GetAuthorNames(authorIDs)
    if err != nil {
        utils.FailFrom(c, err, "Failed to get comment authors")
        return
    }

    commentList := make([]map[string]interface{}, len(comments))
    for i, comment := range comments {
        commentList[i] = map[string]interface{}{
            "comment_id":  comment.UUID,
            "author_id":   comment.AuthorID,
            "author_name": authorNames[comment.AuthorID],
            "content":     comment.Content,
            "created_at":  comment.CreatedAt,
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "comments": commentList,
        "page":     page,
        "limit":    limit,
        "total":    total,
    })
}

func (h *CommentHandler) GetComment(c *gin.Context) {
    commentID := c.Param("comment_id")

    comment, err := h.commentService.GetCommentByID(commentID)
    if err != nil {
        utils.FailFrom(c, err, "Comment not found")
        return
    }

    // 获取作者信息
    author, err := h.commentService.GetUserByID(comment.AuthorID)
    if err != nil {
        utils.FailFrom(c, err, "Failed to get comment author")
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "comment_id": comment.UUID,
        "video_id":   comment.VideoID,
        "author_id":  comment.AuthorID,
        "author_name": author.LoginName,
        "content":    comment.Content,
        "created_at": comment.CreatedAt,
    })
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
        return
    }

    commentID := c.Param("comment_id")

    var req struct {
        Content string `json:"content" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        utils.Fail(c, http.StatusBadRequest, "Invalid request format")
        return
    }

    // Validate content
    if len(req.Content) == 0 || len(req.Content) > 1000 {
        utils.Fail(c, http.StatusBadRequest, "Content must be between 1 and 1000 characters")
        return
    }

    // Get the comment to check ownership
    comment, err := h.commentService.GetCommentByID(commentID)
    if err != nil {
        utils.FailFrom(c, err, "Comment not found")
        return
    }

    if comment.AuthorID != userID.(uint) {
        utils.Fail(c, http.StatusForbidden, "Not authorized to update this comment")
        return
    }

    updates := map[string]interface{}{
        "content": req.Content,
    }

    err = h.commentService.UpdateComment(commentID, updates)
    if err != nil {
        utils.FailFrom(c, err, "Failed to update comment")
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Comment updated successfully"})
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        utils.Fail(c, http.StatusUnauthorized, "Unauthorized")
        return
    }

    commentID := c.Param("comment_id")

    // Get the comment to check ownership
    comment, err := h.commentService.GetCommentByID(commentID)
    if err != nil {
        utils.FailFrom(c, err, "Comment not found")
        return
    }

    if comment.AuthorID != userID.(uint) {
        utils.Fail(c, http.StatusForbidden, "Not authorized to delete this comment")
        return
    }

    err = h.commentService.DeleteComment(commentID)
    if err != nil {
        utils.FailFrom(c, err, "Failed to delete comment")
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}