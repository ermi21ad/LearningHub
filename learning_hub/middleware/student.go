package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func StudentOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists || userRole != "student" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Student access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
