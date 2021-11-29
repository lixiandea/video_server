package main

import (
	"github.com/lixiandea/video_server/entity"
	"github.com/lixiandea/video_server/scheduler"
	"github.com/lixiandea/video_server/scheduler/taskrunner"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	go taskrunner.Start()
	r := scheduler.RegisterHandlers()
	log.Printf("video_dir: ")
	log.Println(filepath.Abs(entity.VIDEO_DIR))
	log.Printf("listen to :10089")
	err := http.ListenAndServe(":10089", r)
	if err != nil {
		log.Fatal("err when listen")
		return
	}
}
