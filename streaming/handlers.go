package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/julienschmidt/httprouter"
)

func UploadVideoHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "文件过大")
		return
	}
	file, _, err := r.FormFile("file") // <form name= "file">
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "获取视频内容错误")
		return
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "读取视频内容错误")
		return
	}
	fn := p.ByName("vid-id")
	err = ioutil.WriteFile(VIDEO_DIR+fn, data, 0666) // path, data, chmod
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "写文件错误")
		return
	}
	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, "upload success")
}

func GetVideoHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	vid := p.ByName("vid-id")
	vl := VIDEO_DIR + vid
	video, err := os.Open(vl)
	defer video.Close()
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "视频路径错误")
		return
	}

	w.Header().Set("Cotent-Type", "video/mp4")
	http.ServeContent(w, r, "", time.Now(), video)

}

func TestPageHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fullpath, err := filepath.Abs(TEMPLATE_PATH + "upload.html")
	if err != nil {
		log.Fatalf("get full path fail")
	}
	log.Panicf("template path: %s", fullpath)
	t, err := template.ParseFiles(TEMPLATE_PATH + "upload.html")
	if err != nil {
		log.Printf("err parse html template")
	}
	t.Execute(w, nil)
}
