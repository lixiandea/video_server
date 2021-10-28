package main

import (
	"github.com/lixiandea/video_server/scheduler"
	"github.com/lixiandea/video_server/scheduler/taskrunner"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisterHandler() *httprouter.Router {
	router := httprouter.New()

	router.GET("/video-delete-record/:vid-id", scheduler.VideoDelRecHandler)
	return router
}

func main() {
	go taskrunner.Start()
	r := RegisterHandler()
	log.Printf("listen to :10089")
	err := http.ListenAndServe(":10089", r)
	if err != nil {
		log.Fatal("err when listen")
		return
	}
}
