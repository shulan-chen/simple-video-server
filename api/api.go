package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type middleWareHandler struct {
	r *httprouter.Router
}

func NewMiddleWareHandler(r *httprouter.Router) *middleWareHandler {
	m := middleWareHandler{}
	m.r = r
	return &m
}

func (m *middleWareHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// do some logging
	// do some auth
	validateUserSession(req)
	m.r.ServeHTTP(w, req)
}

func RegisterHandlers() *httprouter.Router {
	router := httprouter.New()
	router.POST("/user", CreateUser)
	router.POST("/user/:user_name", Login)
	router.GET("/user/:user_name", GetUserInfo)

	router.POST("/user/:user_name/videos", AddNewVideo)
	router.GET("/user/:user_name/videos", ListUserAllVideos)
	router.DELETE("/user/:user_name/videos/:vid", DeleteVideoInfo)

	router.POST("/videos/:vid/comments", PostComments)
	router.GET("/videos/:vid/comments", ListComments)
	return router
}

func Start() {
	r := RegisterHandlers()
	m := NewMiddleWareHandler(r)
	http.ListenAndServe(":8000", m)
}
