package utils

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterSwagger 注册Swagger路由
func RegisterSwagger(router *gin.Engine) {
	// Swagger文档路由
	// 访问: http://localhost:8000/swagger/index.html
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
