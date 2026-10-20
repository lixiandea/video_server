package metrics

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics collectors
var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpRequestBodySize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_body_size_bytes",
			Help:    "HTTP request body size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseBodySize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_body_size_bytes",
			Help:    "HTTP response body size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	// Business metrics
	activeUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users_total",
			Help: "Number of active users",
		},
	)

	videoUploadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "video_uploads_total",
			Help: "Total number of video uploads",
		},
	)

	videoViewsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "video_views_total",
			Help: "Total number of video views",
		},
	)

	commentsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "comments_total",
			Help: "Total number of comments",
		},
	)

	// Transcode queue metrics
	transcodeQueueLength = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transcode_queue_length",
			Help: "Current length of transcode queue",
		},
	)

	transcodeQueuePending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transcode_queue_pending",
			Help: "Number of pending transcode tasks",
		},
	)

	transcodeQueueProcessing = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transcode_queue_processing",
			Help: "Number of processing transcode tasks",
		},
	)

	transcodeQueueCompleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "transcode_queue_completed_total",
			Help: "Total number of completed transcode tasks",
		},
	)

	transcodeQueueFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "transcode_queue_failed_total",
			Help: "Total number of failed transcode tasks",
		},
	)

	transcodeTaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transcode_task_duration_seconds",
			Help:    "Transcode task duration in seconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 10),
		},
		[]string{"quality", "status"},
	)

	// System metrics
	goRoutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_routines",
			Help: "Number of goroutines",
		},
	)

	memoryAllocBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_alloc_bytes",
			Help: "Current memory allocation in bytes",
		},
	)

	// Rate limiter metrics
	rateLimiterEntries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "rate_limiter_entries",
			Help: "Number of per-client rate limiters currently tracked",
		},
	)

	// Database metrics
	dbConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of open database connections",
		},
	)

	dbQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"table", "operation"},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"table", "operation"},
	)

	// Authentication metrics
	authSuccessTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_success_total",
			Help: "Total number of successful authentications",
		},
	)

	authFailureTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_failure_total",
			Help: "Total number of failed authentications",
		},
	)

	uploadRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "upload_rejected_total",
			Help: "Total upload requests rejected by policy",
		},
		[]string{"reason"},
	)

	reviewBacklog = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "video_review_backlog",
			Help: "Number of videos waiting for moderation",
		},
	)

	mediaReplicationPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "media_replication_pending",
			Help: "Number of active videos waiting for backup replication",
		},
	)

	mediaReplicationLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "media_replication_lag_seconds",
			Help: "Age of the oldest unreplicated active video",
		},
	)

	auditDroppedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "audit_logs_dropped_total",
			Help: "Audit records dropped because the async queue was full",
		},
	)
)

// MetricsMiddleware collects HTTP metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics after request is processed
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Increment request counter
		httpRequestsTotal.WithLabelValues(c.Request.Method, endpoint, status).Inc()

		// Record request duration
		httpRequestDuration.WithLabelValues(c.Request.Method, endpoint).Observe(duration)

		// Record request/response sizes
		if c.Request.ContentLength > 0 {
			httpRequestBodySize.WithLabelValues(c.Request.Method, endpoint).Observe(float64(c.Request.ContentLength))
		}

		if c.Writer.Size() > 0 {
			httpResponseBodySize.WithLabelValues(c.Request.Method, endpoint).Observe(float64(c.Writer.Size()))
		}
	}
}

// RegisterMetricsEndpoint registers the /metrics endpoint.
//
// 指标包含内部拓扑与调用量，不应对外公开，因此按两种方式之一保护：
//  1. 配置了 token 时要求 `Authorization: Bearer <token>`（推荐，Prometheus 侧用
//     `authorization.credentials_file` 提供）；
//  2. 未配置 token 时只允许来自回环 / 私有网段的抓取，公网来源返回 403。
//
// 判断使用 TCP 源地址而非 gin 的 ClientIP：后者信任 X-Forwarded-For，可被伪造。
func RegisterMetricsEndpoint(r *gin.Engine, token string) {
	handler := gin.WrapH(promhttp.Handler())

	r.GET("/metrics", func(c *gin.Context) {
		if token != "" {
			expected := "Bearer " + token
			provided := c.GetHeader("Authorization")
			if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
				c.Header("WWW-Authenticate", "Bearer")
				utils.Fail(c, http.StatusUnauthorized, "Metrics token required")
				return
			}
			handler(c)
			return
		}

		if !isPrivateRemote(c.Request.RemoteAddr) {
			utils.Fail(c, http.StatusForbidden, "Metrics endpoint is restricted to private networks")
			return
		}
		handler(c)
	})
}

// isPrivateRemote 判断 TCP 源地址是否属于回环 / 私有 / 链路本地网段。
// Docker Compose 内网的 Prometheus（172.16/12）因此默认可正常抓取。
func isPrivateRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// SetRateLimiterEntries 上报当前被跟踪的限流器数量，便于观察内存增长。
func SetRateLimiterEntries(count int) {
	rateLimiterEntries.Set(float64(count))
}

// Business metric helpers
func IncActiveUsers() {
	activeUsers.Inc()
}

func DecActiveUsers() {
	activeUsers.Dec()
}

func IncVideoUploads() {
	videoUploadsTotal.Inc()
}

func IncVideoViews() {
	videoViewsTotal.Inc()
}

func IncComments() {
	commentsTotal.Inc()
}

func IncAuthSuccess() {
	authSuccessTotal.Inc()
}

func IncAuthFailure() {
	authFailureTotal.Inc()
}

func IncUploadRejected(reason string) {
	uploadRejectedTotal.WithLabelValues(reason).Inc()
}

func SetReviewBacklog(count int64) {
	reviewBacklog.Set(float64(count))
}

func SetMediaReplicationStats(pending int64, oldest time.Time) {
	mediaReplicationPending.Set(float64(pending))
	if pending > 0 && !oldest.IsZero() {
		mediaReplicationLag.Set(time.Since(oldest).Seconds())
	} else {
		mediaReplicationLag.Set(0)
	}
}

func IncAuditDropped() {
	auditDroppedTotal.Inc()
}

func SetDBConnections(count float64) {
	dbConnections.Set(count)
}

// Transcode queue metrics helpers
func SetTranscodeQueueStats(length, pending, processing int) {
	transcodeQueueLength.Set(float64(length))
	transcodeQueuePending.Set(float64(pending))
	transcodeQueueProcessing.Set(float64(processing))
}

func IncTranscodeCompleted() {
	transcodeQueueCompleted.Inc()
}

func IncTranscodeFailed() {
	transcodeQueueFailed.Inc()
}

func ObserveTranscodeDuration(duration time.Duration, quality, status string) {
	transcodeTaskDuration.WithLabelValues(quality, status).Observe(duration.Seconds())
}

func IncDBQueries(table, operation string) {
	dbQueriesTotal.WithLabelValues(table, operation).Inc()
}

func ObserveDBQueryDuration(table, operation string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(table, operation).Observe(duration.Seconds())
}

// System metrics collector (should be called periodically)
func CollectSystemMetrics() {
	// This would typically be called from a goroutine that runs periodically
	// For now, we'll just collect basic runtime metrics

	// Collect goroutine count
	// goRoutines.Set(float64(runtime.NumGoroutine()))

	// Collect memory stats
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// memoryAllocBytes.Set(float64(m.Alloc))
}
