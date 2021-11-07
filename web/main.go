package main

import (
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

func RegisterHandler() *httprouter.Router {
	// bind handlers
	router := httprouter.New()
	router.GET("/", HomeHandler)
	router.POST("/", HomeHandler)
	router.GET("/userhome", UserHomeHandler)
	router.POST("/userhome", UserHomeHandler)
	router.POST("/api", APIHandler)
	//static file bind
	router.ServeFiles("/statics/*filepath", http.Dir("./template"))

	router.POST("/vedio/:vid", proxyHandler)
	return router
}

func main() {
	r := RegisterHandler()
	log.Printf("web service listen to 10090")
	http.ListenAndServe(":10090", r)
}
