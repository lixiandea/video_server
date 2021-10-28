package streaming

import (
	"github.com/lixiandea/video_server/streaming"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisteryHandlers() *httprouter.Router {
	router := httprouter.New()
	router.GET("/videos/:vid-id", streaming.GetVideoHandler)
	router.POST("/upload/:vid-id", streaming.UploadVideoHandler)
	router.GET("/video/testpage", streaming.TestPageHandler)
	return router
}

func main() {
	r := RegisteryHandlers()
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
