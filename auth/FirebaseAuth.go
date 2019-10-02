package auth

import (
	"github.com/gin-gonic/gin"
	"log"
	"strings"
)

func FirebaseAuthMiddleware() gin.HandlerFunc {
	authService, _ := GetAuthService()
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		token := ""
		if authHeader != "" {
			// get token from Authorization header
			token = strings.TrimPrefix(authHeader,"Bearer ")
		} else {
			// try to get the query param accessToken
			token = c.Query("accessToken")
		}

		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Missing JWT Token"})
		} else {
			log.Println("JWT Token", token)
			user, err := authService.GetUserFromToken(token)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{"err": err})
			} else {
				c.Set("user", user)
				c.Set("jwtToken", token)
				c.Next()
			}
		}
	}

}
