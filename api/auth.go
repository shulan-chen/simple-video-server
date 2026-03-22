package api

import (
	"net/http"
)

var HEADER_FILED_SESSION = "X-Session-Id"
var HEADER_FILED_UNAME = "X-User-Name"
var HEADER_FILED_UID = "X-User-Id"

// ValidateUser 检查请求头中是否有用户信息（由中间件设置）
func ValidateUser(w http.ResponseWriter, req *http.Request) bool {
	uname := req.Header.Get(HEADER_FILED_UNAME)
	if uname == "" {
		// 如果到这里还没有用户信息，说明中间件有问题
		return false
	}
	return true
}
