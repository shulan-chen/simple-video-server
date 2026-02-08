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

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if !m.l.GetConn() {
		sendErrorResponse(w, http.StatusTooManyRequests, "too many requests")
		return
	}
	m.r.ServeHTTP(w, req)
	m.l.Release()
}

func RegisterHandlers() *httprouter.Router {
	router := httprouter.New()
	router.GET("/videos/:vid-id", streamOssHandler)
	router.POST("/videos/upload/:vid-id", uploadOssHandler)
	router.GET("/testVideoPage", testPageHandler)
	return router
}

func Start() {
	router := RegisterHandlers()
	m := NewMiddleWareHandler(router, 10)
	http.ListenAndServe(":9090", m)
}


/* podman run -d \
  --name mysql \
  --restart=unless-stopped \
  -p 0.0.0.0:3306:3306 \
  -e MYSQL_ROOT_PASSWORD=123456 \
  -e TZ=Asia/Shanghai \
  -v /opt/containers/mysql/data:/var/lib/mysql:Z \
  -v /opt/containers/mysql/logs:/var/log/mysql:Z \
  docker.io/library/mysql:8.0

    -v /opt/containers/mysql/my.cnf:/etc/mysql/conf.d/my.cnf:Z \ */