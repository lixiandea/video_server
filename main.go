package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisteryHandlers() *httprouter.Router {
	router := httprouter.New()
	router.POST("/user", CreateUser)
	router.POST("/user/:username", Login)
	router.GET("/user/:username/videos", ListVideos)
	router.GET("/user/:username/videos/:vid-id", GetVideo)
	router.GET("/videos/:vid-id/comments", GetComments)
	router.POST("/videos/:vid-id/comments", UpdateComments)
	router.DELETE("/videos/:videoid/comments/:comment-id", DeleteComment)
	return router
}

func main() {
	r := RegisteryHandlers()
	http.ListenAndServe(":10086", r)
}

// handler -> validate { request validate && user validate} -> business logic -> response
// validate:
