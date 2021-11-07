package main

import (
	"bytes"
	"encoding/json"
	"github.com/lixiandea/video_server/entity"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

var client *http.Client

func init() {
	client = &http.Client{}
}
func request(w http.ResponseWriter, r *http.Request, body *ApiBody) {
	var resp *http.Response
	var err error
	switch body.Method {
	case http.MethodPost:
		req, _ := http.NewRequest("POST", body.Url, bytes.NewBuffer([]byte(body.ReqBody)))
		req.Header = r.Header
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("get failed : %v", err)
			return
		}
		SendNormalResponse(w, resp)
	case http.MethodGet:
		req, _ := http.NewRequest("GET", body.Url, nil)
		req.Header = r.Header
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("Post failed : %v", err)
			return
		}
		SendNormalResponse(w, resp)
	case http.MethodDelete:
		req, _ := http.NewRequest("DELETE", body.Url, nil)
		req.Header = r.Header
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("Delete failed : %v", err)
			return
		}
		SendNormalResponse(w, resp)
	default:
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "api bad request")
	}
}

func SendNormalResponse(w http.ResponseWriter, resp *http.Response) {
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		re, _ := json.Marshal(entity.ErrorInternalFaults)
		io.WriteString(w, string(re))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(resp.StatusCode)
	io.WriteString(w, string(res))

}
