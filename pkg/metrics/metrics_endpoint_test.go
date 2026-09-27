package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newMetricsRouter(token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterMetricsEndpoint(r, token)

	return r
}

func doRequest(t *testing.T, r *gin.Engine, remoteAddr, authorization string) int {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = remoteAddr
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w.Code
}

// 配置了令牌时必须携带正确的 Bearer 头，避免 /metrics 直接对外暴露。
func TestMetricsEndpointRequiresToken(t *testing.T) {
	r := newMetricsRouter("s3cret-token")

	if got := doRequest(t, r, "203.0.113.7:5555", ""); got != http.StatusUnauthorized {
		t.Fatalf("missing token: expected 401, got %d", got)
	}
	if got := doRequest(t, r, "203.0.113.7:5555", "Bearer wrong"); got != http.StatusUnauthorized {
		t.Fatalf("wrong token: expected 401, got %d", got)
	}
	if got := doRequest(t, r, "203.0.113.7:5555", "Bearer s3cret-token"); got != http.StatusOK {
		t.Fatalf("correct token: expected 200, got %d", got)
	}
}

// 未配置令牌时退化为「仅内网可抓取」：容器内网的 Prometheus 属私有网段，
// 公网来源必须被拒绝。
func TestMetricsEndpointRestrictsPublicClients(t *testing.T) {
	r := newMetricsRouter("")

	if got := doRequest(t, r, "203.0.113.7:5555", ""); got != http.StatusForbidden {
		t.Fatalf("public client: expected 403, got %d", got)
	}
	if got := doRequest(t, r, "172.18.0.5:5555", ""); got != http.StatusOK {
		t.Fatalf("docker network client: expected 200, got %d", got)
	}
	if got := doRequest(t, r, "127.0.0.1:5555", ""); got != http.StatusOK {
		t.Fatalf("loopback client: expected 200, got %d", got)
	}
}
