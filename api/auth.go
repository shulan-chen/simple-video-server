package api

import (
	"net/http"
	api "video-server/api/defs"
	"video-server/api/session"
)

var HEADER_FILED_SESSION = "X-Session-Id"
var HEADER_FILED_UNAME = "X-User-Name"

func validateUserSession(req *http.Request) bool {
	sid := req.Header.Get(HEADER_FILED_SESSION)
	if sid == "" {
		return false
	}
	userName, ok := session.IsSessionExpired(sid)
	if !ok {
		return false
	}
	req.Header.Add(HEADER_FILED_UNAME, userName)
	return true
}

func ValidateUser(w http.ResponseWriter, req *http.Request) bool {
	uname := req.Header.Get(HEADER_FILED_UNAME)
	if uname == "" {
		sendErrorResponse(w, api.ErrorNotAuthUser)
		return false
	}
	return true
}
