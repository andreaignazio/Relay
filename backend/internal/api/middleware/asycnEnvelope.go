package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func AsyncEnvelope() gin.HandlerFunc {
	return func(c *gin.Context) {
		messageIDStr := c.GetHeader("X-Message-ID")
		if messageIDStr == "" {
			c.AbortWithStatusJSON(400, gin.H{"error": "Missing X-Message-ID header"})
			return
		}
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "Invalid X-Message-ID header"})
			return
		}
		c.Set("X-Message-ID", messageID)

		actionKeyStr := c.GetHeader("X-Action-Key")
		if actionKeyStr == "" {
			c.AbortWithStatusJSON(400, gin.H{"error": "Missing X-Action-Key header"})
			return
		}
		c.Set("X-Action-Key", actionKeyStr)

		c.Next()
	}

}
