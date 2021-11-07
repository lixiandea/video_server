package main

import (
	"github.com/lixiandea/video_server/scheduler"
	"github.com/lixiandea/video_server/scheduler/taskrunner"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	go taskrunner.Start()
	r := scheduler.RegisterHandlers()
	log.Printf("listen to :10089")
	err := http.ListenAndServe(":10089", r)
	if err != nil {
		log.Fatal("err when listen")
		return
	}
}
