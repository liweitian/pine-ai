package v1

import (
	"encoding/json"
	"net/http"
	"pine-ai/dto"
	"pine-ai/global"
	"pine-ai/service"

	"github.com/gin-gonic/gin"
)

func RegisterModelAPI(c *gin.Context) {
	var req dto.RegisterModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := service.ModelRegistry.Register(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "model registered",
		"model":   req.ModelName,
		"version": req.Version,
	})
}

func ListModelsAPI(c *gin.Context) {
	models := service.ModelRegistry.List()
	c.JSON(http.StatusOK, models)
}

func UpdateModelAPI(c *gin.Context) {
	var req dto.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := service.ModelRegistry.Update(req.ModelName, req.Version, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func DeleteModelAPI(c *gin.Context) {
	modelName := c.Param(global.ModelName)
	version := c.Param(global.Version)

	if err := service.ModelRegistry.Delete(modelName, version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func InferAPI(c *gin.Context) {
	var req dto.InferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE headers for streaming tokens to frontend.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	writeSSE := func(event string, payload any) {
		b, err := json.Marshal(payload)
		if err != nil {
			return
		}
		c.Writer.WriteString("event: " + event + "\n")
		c.Writer.WriteString("data: " + string(b) + "\n\n")
		flusher.Flush()
	}

	err := service.InferService.StreamInfer(
		c.Request.Context(),
		req.Model,
		req.Version,
		req.Input,
		func(token string) error {
			writeSSE("token", gin.H{"content": token})
			return nil
		},
	)
	if err != nil {
		writeSSE("error", gin.H{"message": err.Error()})
	}
	writeSSE("done", gin.H{"ok": err == nil})
}
