package router

import (
	v1 "pine-ai/router/api/v1"

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
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	apiV1 := r.Group("/api/v1/pine-ai")
	{
		apiV1.POST("/models", v1.RegisterModelAPI)
		// apiV1.GET("/models", v1.ListModelsAPI)
		// apiV1.PUT("/models/:name/version/:v", v1.UpdateModelAPI)
		// apiV1.POST("/infer", v1.InferAPI)
	}

	return r
}
