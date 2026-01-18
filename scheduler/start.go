package scheduler

import (
	"net/http"
	"video-server/scheduler/taskrunner"

	"github.com/julienschmidt/httprouter"
)

func RegisterHandlers() *httprouter.Router {
	router := httprouter.New()
	router.GET("/video-delete-record/:vid-vid", vidDelRecHandler)
	return router
}

func Start() {
	go taskrunner.Start()
	r := RegisterHandlers()
	http.ListenAndServe(":9091", r)
}
