package router

import (
	"net/http"

	"pine-ai/config"
	"pine-ai/router/api"
	"pine-ai/service"

	"github.com/gin-gonic/gin"
)

func New(cfg config.Config) *gin.Engine {
	r := gin.Default()

	// Model托管（注册/更新/查询）
	modelRegistry := service.NewRegistry()
	api.RegisterModelAPI(r, modelRegistry)

	// 推理服务（SSE流式输出）
	inferService := service.NewInferService(cfg, modelRegistry)
	api.RegisterInferAPI(r, inferService)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	return r
}
