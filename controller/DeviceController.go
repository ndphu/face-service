package controller

import (
	"framed-mqtt-client/db"
	"framed-mqtt-client/model"
	"github.com/gin-gonic/gin"
)

func DeviceController(r *gin.RouterGroup) {
	r.GET("/device/:deviceId/capture", func(c *gin.Context) {
		device := model.Device{
			DeviceId: c.Param("deviceId"),
		}

		if raw, err := device.Capture(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.Header("Content-Type", "image/jpg")
			c.Writer.Write(raw)
		}
	})

	r.GET("/device/:deviceId/detectFaces", func(c *gin.Context) {
		device := model.Device{
			DeviceId: c.Param("deviceId"),
		}

		if resp, err := device.DetectFaces(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, resp)
		}
	})

	r.GET("/device/:deviceId/recognizeFaces", func(c *gin.Context) {
		device := model.Device{
			DeviceId: c.Param("deviceId"),
		}

		if resp, err := device.RecognizeFaces(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, resp)
		}
	})

	r.GET("/device/:deviceId/reloadSamples", func(c *gin.Context) {
		device := model.Device{
			DeviceId: c.Param("deviceId"),
		}

		if err := device.ReloadSamples(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{})
		}
	})

	r.GET("/device/:deviceId/faceInfos", func(c *gin.Context) {
		var faces []model.Face
		if err := dao.Collection("face").Find(nil).All(&faces); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, faces)
		}
	})

}
