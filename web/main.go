package main

import (
	"log"
	"net/http"
)

func main() {
	r := RegisterHandlers()
	log.Printf("web service listen to 10090")
	http.ListenAndServe(":10090", r)
}
