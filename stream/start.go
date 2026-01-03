package stream

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type middleWareHandler struct {
	r *httprouter.Router
	l *ConnLimiter
}

func NewMiddleWareHandler(r *httprouter.Router, connLimitNumber int) *middleWareHandler {
	return &middleWareHandler{
		r: r,
		l: NewConnLimiter(connLimitNumber),
	}
}

func (m *middleWareHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !m.l.GetConn() {
		sendErrorResponse(w, http.StatusTooManyRequests, "too many requests")
		return
	}
	m.r.ServeHTTP(w, req)
	m.l.Release()
}

func RegisterHandlers() *httprouter.Router {
	router := httprouter.New()
	router.GET("/videos/:vid-id", streamHandler)
	router.POST("/videos/upload/:vid-id", uploadHandler)
	router.GET("/testVideoPage", testPageHandler)
	return router
}

func Start() {
	router := RegisterHandlers()
	m := NewMiddleWareHandler(router, 2)
	http.ListenAndServe(":9090", m)
}
