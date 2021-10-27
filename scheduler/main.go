package main

import (
	"log"
	"net/http"
	"scheduler/taskrunner"

	"github.com/julienschmidt/httprouter"
)

func RegisterHandler() *httprouter.Router {
	router := httprouter.New()

	router.GET("/video-delete-record/:vid-id", VideoDelRecHandler)
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
