package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func InstructorOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists || userRole != "instructor" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Instructor access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
