package v1

import (
	"net/http"
	"pine-ai/dto"
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

	c.JSON(http.StatusCreated, gin.H{
		"message": "model registered",
		"model":   req.ModelName,
		"version": req.Version,
	})
}
