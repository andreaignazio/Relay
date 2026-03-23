package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		shouldGenerate := false
		var TraceID uuid.UUID
		traceIDStr := c.GetHeader("X-Trace-ID")
		if traceIDStr == "" {
			shouldGenerate = true
		} else {
			parsedID, err := uuid.Parse(traceIDStr)
			if err != nil {
				shouldGenerate = true
			} else {
				TraceID = parsedID
			}
		}

		if shouldGenerate {
			TraceID = generateTraceID()
		}

		c.Set("TraceID", TraceID)

		c.Next()
	}
}

func generateTraceID() uuid.UUID {
	return uuid.New()
}
