package web

import (
	"github.com/gin-gonic/gin"
)

func RegisterHandlers() *gin.Engine {
	router := gin.Default()

	router.LoadHTMLGlob("templates/*.html")
	router.Static("/statics/", "./templates")

	router.GET("/", homeHandler)
	router.POST("/", homeHandler)
	router.GET("/userhome", userHomeHandler)
	router.POST("/userhome", userHomeHandler)
	//API 透传路由
	router.POST("/api", apiHandler)
	router.GET("/videos/:vid-id", proxyVideoViewHandler)
	router.POST("/videos/upload/:vid-id", proxyUploadHandler)

	return router
}

func Start() {
	r := RegisterHandlers()
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
