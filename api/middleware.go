package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Middleware struct {
	r *httprouter.Router
}

func NewMiddleware(r *httprouter.Router) http.Handler {
	m := Middleware{}
	m.r = r
	return m
}

func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// do auth
	validateUserSession(r)

	m.r.ServeHTTP(w, r)
}
