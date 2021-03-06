package scheduler

import (
	"github.com/lixiandea/video_server/dbops"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func videoDelRecHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

func RegisterHandlers() *httprouter.Router {
	router := httprouter.New()

	router.GET("/video-delete-record/:vid-id", videoDelRecHandler)
	return router
}
