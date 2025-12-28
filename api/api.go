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
	return router
}

func Start() {
	r := RegisterHandlers()
	m := NewMiddleWareHandler(r)
	http.ListenAndServe(":8080", m)
}
