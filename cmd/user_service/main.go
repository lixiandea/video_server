package main

import (
	"github.com/lixiandea/video_server/user_service"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	r := user_service.RegisteryHandlers()
	// mh := NewMiddleware(r)
	log.Printf("streaming listen to 10087")
	http.ListenAndServe(":10087", r)
}

// handler -> validate { request validate && user validate} -> business logic -> response
// validate: session
// main -> middleware -> defs -> handlers -> dbops -> response
