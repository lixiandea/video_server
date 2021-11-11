package user_service

import (
	"log"
	"net/http"
	"testing"
)

func TestUserService(t *testing.T) {
	r := RegisteryHandlers()
	// mh := NewMiddleware(r)
	log.Printf("streaming listen to 10087")
	http.ListenAndServe(":10087", r)
}
