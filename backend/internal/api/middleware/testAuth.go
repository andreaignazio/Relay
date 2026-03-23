package middleware

import (
	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
		c.Set("UserID", userID)
		c.Next()
	}
}
