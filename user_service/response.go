package user_service

import (
	"encoding/json"
	"github.com/lixiandea/video_server/entity"
	"io"
	"net/http"
)

func SendErrorResponse(w http.ResponseWriter, errResp entity.ErrResponse) {
	w.WriteHeader(errResp.HttpSc)
	resStr, _ := json.Marshal(&errResp.Error)
	io.WriteString(w, string(resStr))
}

func SendNormalResponse(w http.ResponseWriter, resp string, sc int) {
	w.WriteHeader(sc)
	io.WriteString(w, resp)
}
