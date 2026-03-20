package router

import (
	"net/http"
	v1 "pine-ai/router/api/v1"
	"pine-ai/router/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AllowHeaders = []string{"*"}
	r := gin.New()
	r.Use(cors.New(config))
	r.Use(gin.Recovery())
	r.Use(middleware.TraceAndLog())

	apiV1 := r.Group("/api/v1/pine-ai")
	{
		apiV1.POST("/models", v1.RegisterModelAPI)
		apiV1.GET("/models", v1.ListModelsAPI)
		apiV1.PUT("/models/:name/version/:v", v1.UpdateModelAPI)
		apiV1.POST("/infer", v1.InferAPI)
		apiV1.GET("/hello", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"message": "hello"})
		})
	}
	return r
}
