package main

import (
	"log"
	"net/http"
)

func main() {
	r := RegisterHandlers()
	log.Printf("web service listen to 10090")
	err := http.ListenAndServe(":10090", r)
	if err != nil {
		return
	}
}
