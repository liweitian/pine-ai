package api

import (
	"encoding/json"
	"net/http"
	"pine-ai/service"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterModelAPI(r *gin.Engine, registry *service.Registry) {
	r.POST("/models", func(c *gin.Context) {
		var req service.RegisterModelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := registry.Register(req); err != nil {
			// 注册重复按 409，其它错误按 400。
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "already") {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "model registered"})
	})

	r.GET("/models", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"models": registry.List()})
	})

	r.PUT("/models/:name/version/:v", func(c *gin.Context) {
		var req service.UpdateModelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := registry.Update(c.Param("name"), c.Param("v"), req); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "model version updated"})
	})
}

func RegisterInferAPI(r *gin.Engine, infer *service.InferService) {
	r.POST("/infer", func(c *gin.Context) {
		var req service.InferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "response writer does not support flushing"})
			return
		}

		send := func(event string, data any) {
			b, err := json.Marshal(data)
			if err != nil {
				return
			}
			// SSE message format:
			// event: <event>\n
			// data: <json>\n\n
			_, _ = c.Writer.WriteString("event: " + event + "\n" + "data: " + string(b) + "\n\n")
			flusher.Flush()
		}

		err := infer.StreamInfer(
			c.Request.Context(),
			req.Model,
			req.Version,
			req.Input,
			func(meta service.InferMeta) error {
				send("meta", meta)
				return nil
			},
			func(token string) error {
				send("token", gin.H{"content": token})
				return nil
			},
		)
		if err != nil {
			send("error", gin.H{"message": err.Error()})
		}
		send("done", gin.H{"ok": err == nil})
	})
}
