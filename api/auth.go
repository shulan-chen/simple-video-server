package api

import (
	"net/http"
	"strings"
	api "video-server/api/defs"
	"video-server/api/session"
)

var HEADER_FILED_SESSION = "X-Session-Id"
var HEADER_FILED_UNAME = "X-User-Name"

func validateUserSession(w http.ResponseWriter, req *http.Request) (needNextStep bool) {
	// register and login do not need session validation
	if req.URL.Path == "/user" && req.Method == "POST" {
		return true
	}
	if req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/user/") {
		rest := strings.TrimPrefix(req.URL.Path, "/user/")
		// 如果剩余部分不包含 "/"，说明是 /user/:username 形式
		if !strings.Contains(rest, "/") {
			return true
		}
	}
	//检查有没有X-Session-Id头，没有/session过期都视为身份不合法
	sid := req.Header.Get(HEADER_FILED_SESSION)
	if sid == "" {
		sendErrorResponse(w, api.ErrorNotAuthUser)
		return false
	}
	userName, expired := session.IsSessionExpired(sid)
	if expired {
		sendErrorResponse(w, api.ErrorUserloginStatusExpired)
		return false
	}
	//session存在且没过期，就为请求加上X-User-Name头，后续的就只用校验是否存在该Header就行
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
