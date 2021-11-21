package main

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/lixiandea/video_server/entity"
	"github.com/lixiandea/video_server/user_service"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type HomePage struct {
	Name string
}

type UserPage struct {
	Name string
}

func apiHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if r.Method != http.MethodPost {
		user_service.SendErrorResponse(w, entity.ErrorMethodError)
		return
	}
	//log.Println("get api request")
	res, _ := ioutil.ReadAll(r.Body)
	apiBody := &ApiBody{}
	if err := json.Unmarshal(res, apiBody); err != nil {
		log.Printf("parse err:%v", err)
		user_service.SendErrorResponse(w, entity.ErrorRequestBodyParseFailed)
		return
	}

	request(w, r, apiBody)
	defer r.Body.Close()
}

func userHomeHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	cname, err1 := r.Cookie("username")
	_, err2 := r.Cookie("session")
	if err1 != nil || err2 != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// user login via home page
	fname := r.FormValue("username")
	var u *UserPage

	if len(cname.Value) != 0 {
		u = &UserPage{Name: cname.Value}
	} else if len(fname) != 0 {
		u = &UserPage{Name: fname}
	}

	t, err := template.ParseFiles("../template/userhome.html")
	if err != nil {
		log.Printf("parse user home html file failed: %v", err)
		return
	}
	t.Execute(w, u)
}

func homeHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	cname, err1 := r.Cookie("username")
	sid, err2 := r.Cookie("session")
	if err1 != nil || err2 != nil {
		h := &HomePage{Name: "lixiande"}
		t, err := template.ParseFiles("../template/home.html")
		if err != nil {
			log.Printf("parse home html file failed: %v", err)
			return
		}
		t.Execute(w, h)
		return
	}
	if len(cname.Value) != 0 && len(sid.Value) != 0 {
		http.Redirect(w, r, "/userhome", http.StatusFound)
		return
	}
}
func proxyHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	u, _ := url.Parse("http://127.0.0.1:10088")
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(w, r)
}

func RegisterHandlers() *httprouter.Router {
	// bind handlers
	router := httprouter.New()
	router.GET("/", homeHandler)
	router.POST("/", homeHandler)
	router.GET("/userhome", userHomeHandler)
	router.POST("/userhome", userHomeHandler)
	router.POST("/api", apiHandler)
	//static file bind
	router.ServeFiles("/statics/*filepath", http.Dir("../template"))

	router.POST("/video/:vid", proxyHandler)
	return router
}
