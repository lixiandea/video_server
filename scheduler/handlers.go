package main

import (
	"net/http"
	"scheduler/dbops"

	"github.com/julienschmidt/httprouter"
)

func VideoDelRecHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	vid := p.ByName("vid-id")
	if len(vid) == 0 {
		sendResponse(w, 400, "vid should not be empty!")
		return
	}
	err := dbops.AddVideoDeletionRecord(vid)
	if err != nil {
		sendResponse(w, 500, "Internal server Error!")
		return
	}
	sendResponse(w, 200, "")

}
