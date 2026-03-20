package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceIDKey = "trace_id"

func TraceAndLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set(TraceIDKey, traceID)
		c.Header("X-Trace-Id", traceID)

		c.Next()

		log.Printf("trace_id=%s method=%s path=%s status=%d latency_ms=%d",
			traceID, c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start).Milliseconds())
	}
}
