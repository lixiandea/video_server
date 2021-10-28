package main

import (
	"encoding/json"
	"github.com/lixiandea/video_server/dbops"
	"github.com/lixiandea/video_server/entity"
	"github.com/lixiandea/video_server/user_service/session"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func CreateUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	res, _ := ioutil.ReadAll(r.Body)
	ubody := &entity.UserCredential{}

	if err := json.Unmarshal(res, ubody); err != nil {
		SendErrorResponse(w, entity.ErrorRequestBodyParseFailed)
		return
	}

	if err := dbops.AddUserCredential(ubody.UserName, ubody.Pwd); err != nil {
		SendErrorResponse(w, entity.ErrorDBError)
	}
	id := session.GenerateNewSessionId(ubody.UserName)
	su := entity.SignedUp{Success: true, SessionID: id}

	if resp, err := json.Marshal(su); err != nil {
		SendErrorResponse(w, entity.ErrorInternalFaults)
	} else {
		SendNormalResponse(w, string(resp), 201)
	}

}

func Login(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func ListVideos(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func GetVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func GetComments(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func DeleteComment(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func UpdateComments(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}
