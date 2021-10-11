package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisteryHandlers() *httprouter.Router {
	router := httprouter.New()
	router.POST("/user", CreateUser)
	router.POST("/user/:user_name", Login)
	return router
}

func main() {
	r := RegisteryHandlers()
	http.ListenAndServe(":10086", r)
}
