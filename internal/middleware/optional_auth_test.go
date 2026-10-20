package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lixiandea/video_server/pkg/auth"
)

type optionalAuthValidator struct {
	err error
}

func (v optionalAuthValidator) ValidateSession(string, uint) error {
	return v.err
}

func TestOptionalAuthMiddlewareAllowsAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OptionalAuthMiddleware(optionalAuthValidator{}))
	router.GET("/", func(c *gin.Context) {
		if _, exists := c.Get("user_id"); exists {
			t.Fatal("anonymous request unexpectedly has user context")
		}
		c.Status(http.StatusOK)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestOptionalAuthMiddlewareSetsValidUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := auth.InitJWT("test-jwt-secret-at-least-32-characters"); err != nil {
		t.Fatal(err)
	}
	token, err := auth.GenerateJWT(42, "tester", "session-1")
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(OptionalAuthMiddleware(optionalAuthValidator{}))
	router.GET("/", func(c *gin.Context) {
		if userID := c.GetUint("user_id"); userID != 42 {
			t.Fatalf("expected user 42, got %d", userID)
		}
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestOptionalAuthMiddlewareFallsBackWhenSessionIsInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := auth.InitJWT("test-jwt-secret-at-least-32-characters"); err != nil {
		t.Fatal(err)
	}
	token, err := auth.GenerateJWT(7, "expired", "session-2")
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.Use(OptionalAuthMiddleware(optionalAuthValidator{err: errors.New("expired")}))
	router.GET("/", func(c *gin.Context) {
		if _, exists := c.Get("user_id"); exists {
			t.Fatal("invalid session unexpectedly has user context")
		}
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}
