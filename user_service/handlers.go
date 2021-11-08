package user_service

import (
	"encoding/json"
	"github.com/lixiandea/video_server/dbops"
	"github.com/lixiandea/video_server/entity"
	"github.com/lixiandea/video_server/user_service/session"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

func createUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

func login(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	res, _ := ioutil.ReadAll(r.Body)
	uBody := &entity.UserCredential{}
	// parse body
	if err := json.Unmarshal(res, uBody); err != nil{
		SendErrorResponse(w, entity.ErrorRequestBodyParseFailed)
		return
	}

	//validate user
	uname := p.ByName("username")
	if uname != uBody.UserName{
		SendErrorResponse(w, entity.ErrorNotAuthUser)
	}

	pwd, err := dbops.GetUserCredential(uBody.UserName)
	if err != nil || len(pwd) == 0|| pwd != uBody.Pwd{
		SendErrorResponse(w, entity.ErrorNotAuthUser)
	}

	id := session.GenerateNewSessionId(uBody.UserName)
	si := &entity.SignedUp{Success: true, SessionID: id}

	if resp, err:= json.Marshal(si); err != nil{
		SendErrorResponse(w, entity.ErrorInternalFaults)
	}else {
		SendNormalResponse(w, string(resp), 200)
	}
}

func listVideos(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if !ValidateUser(w, r){
		return
	}

	uname := p.ByName("username")
	ts, err := strconv.Atoi(strconv.FormatInt(time.Now().UnixNano()/1000000000, 10)
	if  err != nil{
		log.Printf("Get timestamp failed.")
		return
	}

	videoInfos, err := dbops.GetVideoInfos(uname, 0, ts)

	if err != nil{
		log.Printf("Error in list all videos: %s", err)
		return
	}

	VIS :=&entity.VideosInfo{Videos:videoInfos}
	if resp, err := json.Marshal(VIS); err != nil{
		SendErrorResponse(w, entity.ErrorInternalFaults)
	}else {
		SendNormalResponse(w, string(resp), 200)
	}
}

func getVideo(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func getComments(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if !ValidateUser(w,r){
		return
	}

	vid := p.ByName("vid-id")
	comments, err := dbops.GetComments(vid)
	if err != nil{
		log.Printf("Error in ShowComments: %v", err)
		SendErrorResponse(w, entity.ErrorDBError)
		return
	}


}

func deleteComment(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	uname := p.ByName("user_name")
	io.WriteString(w, "login user:"+uname)
}

func updateComments(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if !ValidateUser(w,r){
		return
	}

	reqBody, _ := ioutil.ReadAll(r.Body)

	cbody := &entity.Comment{}
	if err := json.Unmarshal(reqBody, cbody); err !=nil{
		SendErrorResponse(w, entity.ErrorRequestBodyParseFailed)
		return
	}
	vid := p.ByName("vid-id")

	if err:= dbops.AddNewComment(vid, cbody.AuthorId, cbody.Content); err != nil{
		log.Printf("error in update comment: %v", err)
	}else {
		SendNormalResponse(w, "ok", 201)
	}
	SendErrorResponse(w, entity.ErrorTest)
}


func RegisteryHandlers() *httprouter.Router {
	router := httprouter.New()
	router.POST("/user",createUser)
	router.POST("/user/:username", login)
	router.GET("/user/:username/videos", listVideos)
	router.GET("/user/:username/videos/:vid-id", getVideo)
	router.GET("/videos/:vid-id/comments", getComments)
	router.POST("/videos/:vid-id/comments", updateComments)
	router.DELETE("/videos/:videoid/comments/:comment-id", deleteComment)
	return router
}