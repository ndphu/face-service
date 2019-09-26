package main

import (
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

	controller.LabelController(apiGroup)
	controller.ProjectController(apiGroup)
	controller.DeviceController(apiGroup)
	controller.WSController(apiGroup)

	r.Run()
}

