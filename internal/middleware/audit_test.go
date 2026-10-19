package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/pkg/logging"
)

func TestAuditMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logPath := filepath.Join(t.TempDir(), "audit.log")
	if err := logging.InitLogger("info", logPath); err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}
	t.Cleanup(func() { _ = logging.Sync() })

	r := gin.New()
	r.Use(AuditMiddleware())
	r.POST("/api/v1/videos/:video_id/like", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		c.Status(http.StatusNoContent)
	})
	r.GET("/api/v1/videos", func(c *gin.Context) { c.Status(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/videos/abc/like", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/videos", nil))

	if err := logging.Sync(); err != nil {
		t.Fatalf("刷新日志失败: %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志失败: %v", err)
	}
	out := string(data)

	for _, want := range []string{
		`"audit":true`,
		`"method":"POST"`,
		`"route":"/api/v1/videos/:video_id/like"`,
		`"status":204`,
		`"user_id":7`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("审计日志缺少 %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, `"method":"GET"`) {
		t.Errorf("GET 请求不应产生审计事件:\n%s", out)
	}
}
