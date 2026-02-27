package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/services"
	"github.com/lixiandea/video_server/pkg/queue"
)

// TranscodeHandler 转码任务处理器
type TranscodeHandler struct {
	transcodeService *services.TranscodeService
	taskQueue        *queue.TaskQueue
}

// NewTranscodeHandler 创建转码任务处理器
func NewTranscodeHandler(transcodeService *services.TranscodeService, taskQueue *queue.TaskQueue) *TranscodeHandler {
	return &TranscodeHandler{
		transcodeService: transcodeService,
		taskQueue:        taskQueue,
	}
}

// GetTaskStatus 获取转码任务状态
func (h *TranscodeHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID required"})
		return
	}

	task, err := h.taskQueue.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":       task.TaskID,
		"video_id":      task.VideoID,
		"status":        task.Status,
		"progress":      task.Progress,
		"target_format": task.TargetFormat,
		"quality":       task.Quality,
		"error":         task.Error,
		"created_at":    task.CreatedAt,
		"started_at":    task.StartedAt,
		"completed_at":  task.CompletedAt,
	})
}

// GetQueueStats 获取队列统计信息
func (h *TranscodeHandler) GetQueueStats(c *gin.Context) {
	stats, err := h.taskQueue.GetQueueStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_length":      stats.QueueLength,
		"pending_count":     stats.PendingCount,
		"processing_count":  stats.ProcessingCount,
		"completed_count":   stats.CompletedCount,
		"failed_count":      stats.FailedCount,
		"total_tasks":       stats.TotalTasks,
		"worker_name":       "transcode_worker",
	})
}

// ListTasks 列出转码任务
func (h *TranscodeHandler) ListTasks(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 获取所有任务（简化实现）
	ctx := c.Request.Context()
	stats, err := h.taskQueue.GetQueueStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": stats.TotalTasks,
		"page":  page,
		"limit": limit,
		"queue_stats": stats,
	})
}

// RetryTask 重试失败的任务
func (h *TranscodeHandler) RetryTask(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.taskQueue.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if task.Status != queue.TaskStatusFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only retry failed tasks"})
		return
	}

	// 重置任务状态
	task.Status = queue.TaskStatusPending
	task.Error = ""
	task.Progress = 0

	err = h.taskQueue.EnqueueTranscodeTask(c.Request.Context(), task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retry task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task queued for retry",
		"task_id": taskID,
	})
}

// CancelTask 取消任务
func (h *TranscodeHandler) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")

	task, err := h.taskQueue.GetTaskStatus(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if task.Status == queue.TaskStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel completed task"})
		return
	}

	// 标记为失败
	task.Status = queue.TaskStatusFailed
	task.Error = "Cancelled by user"
	h.taskQueue.UpdateTaskStatus(c.Request.Context(), task)

	c.JSON(http.StatusOK, gin.H{
		"message": "Task cancelled",
		"task_id": taskID,
	})
}
