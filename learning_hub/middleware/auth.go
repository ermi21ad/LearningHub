package middleware

import (
	"fmt"
	"learning_hub/pkg/jwt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		fmt.Printf("🔐 Auth Middleware - Path: %s, Header: %s\n", c.Request.URL.Path, authHeader)

		if authHeader == "" {
			fmt.Println("❌ No Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		fmt.Printf("🔐 Token: %s\n", tokenString)

		claims, err := jwt.ValidateToken(tokenString)
		if err != nil {
			fmt.Printf("❌ Token validation failed: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		fmt.Printf("✅ Token validated - UserID: %v, Email: %s\n", claims.UserID, claims.Email)

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}
