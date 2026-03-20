package v1

import (
	"net/http"
	"pine-ai/dto"
	"pine-ai/global"
	"pine-ai/router/middleware"
	"pine-ai/service"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterModelAPI(c *gin.Context) {
	var req dto.RegisterModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_PARAM", err.Error())
		return
	}

	if err := service.ModelRegistry.Register(req); err != nil {
		fail(c, http.StatusBadRequest, "REGISTER_FAILED", err.Error())
		return
	}

	ok(c, gin.H{
		"message": "model registered",
		"model":   req.ModelName,
		"version": req.Version,
	})
}

func ListModelsAPI(c *gin.Context) {
	models := service.ModelRegistry.List()
	ok(c, models)
}

func UpdateModelAPI(c *gin.Context) {
	var req dto.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_PARAM", err.Error())
		return
	}

	if err := service.ModelRegistry.Update(req.ModelName, req.Version, req); err != nil {
		fail(c, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	ok(c, gin.H{"message": "model updated"})
}

func DeleteModelAPI(c *gin.Context) {
	modelName := c.Param(global.ModelName)
	version := c.Param(global.Version)

	if err := service.ModelRegistry.Delete(modelName, version); err != nil {
		fail(c, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	ok(c, gin.H{"message": "model deleted"})
}

func InferAPI(c *gin.Context) {
	var req dto.InferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_PARAM", err.Error())
		return
	}

	// SSE headers for streaming tokens to frontend.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		fail(c, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "streaming unsupported")
		return
	}
	traceID := c.GetString(middleware.TraceIDKey)

	err := service.InferService.StreamInfer(
		c.Request.Context(),
		req.Model,
		req.Version,
		req.Input,
		func(token string) error {
			writeSSE(c, "token", gin.H{"content": token, "trace_id": traceID})
			flusher.Flush()
			return nil
		},
	)
	if err != nil {
		code := "INFER_FAILED"
		msg := err.Error()
		if strings.Contains(msg, "timeout") {
			code = "INFER_TIMEOUT"
		} else if strings.Contains(msg, "no_response") {
			code = "INFER_NO_RESPONSE"
		}
		writeSSE(c, "error", gin.H{"code": code, "message": msg, "trace_id": traceID})
		flusher.Flush()
	}
	writeSSE(c, "done", gin.H{"ok": err == nil, "trace_id": traceID})
	flusher.Flush()
}
