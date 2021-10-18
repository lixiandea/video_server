package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Middleware struct {
	r  *httprouter.Router
	cc *ConnLimiter
}

func NewMiddleware(r *httprouter.Router, cc int) http.Handler {
	m := Middleware{}
	m.r = r
	m.cc = &ConnLimiter{numConn: cc, bucket: make(chan int, cc)}
	return m
}

func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !m.cc.getToken() {
		sendErrorResponse(w, http.StatusTooManyRequests, "连接数量超阈值")
	}
	m.r.ServeHTTP(w, r)
	defer m.cc.releaseToken()
}
