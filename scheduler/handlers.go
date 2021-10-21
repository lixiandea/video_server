package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func VideoDelRecHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	vid := p.ByName("vid-id")

}
