package main

import (
	"face-service/auth"
	"face-service/controller"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)


func main() {

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Content-Length", "X-Requested-With", "Connection", "Upgrade"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}))

	apiGroup := r.Group("/api")
	apiGroup.Use(auth.FirebaseAuthMiddleware())

	controller.LabelController(apiGroup)
	controller.DeskController(apiGroup)
	controller.DeviceController(apiGroup)
	controller.WSController(apiGroup)

	authGroup := r.Group("/api/auth")
	controller.AuthController(authGroup)

	r.Run()
}

