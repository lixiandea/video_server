package main

import (
	"github.com/lixiandea/video_server/user_service"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisteryHandlers() *httprouter.Router {
	router := httprouter.New()
	router.POST("/user", user_service.CreateUser)
	router.POST("/user/:username", user_service.Login)
	router.GET("/user/:username/videos", user_service.ListVideos)
	router.GET("/user/:username/videos/:vid-id", user_service.GetVideo)
	router.GET("/videos/:vid-id/comments", user_service.GetComments)
	router.POST("/videos/:vid-id/comments", user_service.UpdateComments)
	router.DELETE("/videos/:videoid/comments/:comment-id", user_service.DeleteComment)
	return router
}

func main() {
	r := RegisteryHandlers()
	// mh := NewMiddleware(r)
	log.Printf("streaming listen to 10087")
	http.ListenAndServe(":10087", r)
}

// handler -> validate { request validate && user validate} -> business logic -> response
// validate: session
// main -> middleware -> defs -> handlers -> dbops -> response
