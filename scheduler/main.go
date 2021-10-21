package main

import (
	"github.com/julienschmidt/httprouter"
)

func RegisterHandler() *httprouter.Router {
	router := httprouter.New()

	router.GET("/video-delete-record/:vid-id", VideoDelRecHandler)
	return router
}
