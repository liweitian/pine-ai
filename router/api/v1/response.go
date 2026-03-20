package v1

import (
	"encoding/json"
	"net/http"

	"pine-ai/router/middleware"

	"github.com/gin-gonic/gin"
)

type apiResp struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, apiResp{
		Code:    "OK",
		TraceID: c.GetString(middleware.TraceIDKey),
		Data:    data,
	})
}

func fail(c *gin.Context, httpStatus int, code, message string) {
	c.JSON(httpStatus, apiResp{
		Code:    code,
		Message: message,
		TraceID: c.GetString(middleware.TraceIDKey),
	})
}

func writeSSE(c *gin.Context, event string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = c.Writer.WriteString("event: " + event + "\n")
	_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
}
