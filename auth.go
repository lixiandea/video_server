package main

import (
	"net/http"
	"video_server/defs"
	"video_server/session"
)

var HEADER_FIELD_SESSION = "X_Session-Id"

var HEADER_FIELD_UNAME = "X-USER-Name"

func validateUserSession(r *http.Request) bool {
	sid := r.Header.Get(HEADER_FIELD_SESSION)
	if len(sid) == 0 {
		return false
	}

	uname, ok := session.IsExpireSession(sid)
	if ok {
		return false
	}

	r.Header.Add(HEADER_FIELD_UNAME, uname)
	return true
}

func ValidateUser(w http.ResponseWriter, r *http.Request) bool {
	uname := r.Header.Get(HEADER_FIELD_SESSION)

	if len(uname) == 0 {
		SendErrorResponse(w, defs.ErrorRequestBodyParseFailed)
		return false
	}

	return true

}
