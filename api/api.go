package api

import (
	"github.com/gin-gonic/gin"
)

func validateUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateUserSession(c.Writer, c.Request) {
			//c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RegisterHandlers() *gin.Engine {
	router := gin.Default()
	router.Use(validateUserMiddleware())
	router.POST("/user", CreateUser)
	router.POST("/user/:user_name", Login)
	router.GET("/user/:user_name", GetUserInfo)
	router.POST("/user/:user_name/logout", Logout)

	router.POST("/user/:user_name/videos", AddNewVideo)
	router.GET("/user/:user_name/videos", ListUserAllVideos)
	router.DELETE("/user/:user_name/videos/:vid", DeleteVideoInfo)

	router.POST("/videos/:vid/comments", PostComments)
	router.GET("/videos/:vid/comments", ListComments)
	router.GET("/videos", ListAllVideos)
	return router
}

func Start() {
	r := RegisterHandlers()
	r.Run(":8000")
}
