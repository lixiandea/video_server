package main

import (
	"github.com/lixiandea/video_server/streaming"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	r := streaming.RegisteryHandlers()
	mh := streaming.NewMiddleware(r, 200)
	// mh := NewMiddleware(r)
	log.Printf("streaming listen to 10088")
	err := http.ListenAndServe(":10088", mh)
	if err != nil {
		log.Fatal("err listen and serve")
	}
}

// handler -> validate { request validate && user validate} -> business logic -> response
// validate: session
// main -> middleware -> defs -> handlers -> dbops -> response
