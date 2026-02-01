package web

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func RegisterHandlers() *httprouter.Router {
	router := httprouter.New()
	router.GET("/", homeHandler)
	router.POST("/", homeHandler)
	router.GET("/userhome", userHomeHandler)
	router.POST("/userhome", userHomeHandler)
	router.POST("/api", apiHandler)
	router.POST("/videos/:vid-id", proxyVideoViewHandler)
	router.POST("/videos/upload/:vid-id", proxyUploadHandler)

	router.ServeFiles("/statics/*filepath", http.Dir("./templates"))
	return router
}

func Start() {
	http.ListenAndServe(":8080", RegisterHandlers())
}
