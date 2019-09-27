package controller

import (
	"face-service/config"
	"face-service/db"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/swd-commons/model"
	"github.com/ndphu/swd-commons/service"
	"strconv"
)

func DeviceController(r *gin.RouterGroup) {
	r.GET("/device/:deviceId", func(c *gin.Context) {
		var device model.Device
		if err := dao.Collection("device").Find(bson.M{"deviceId": c.Param("deviceId")}).One(&device); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, device)
		}
	})

	r.GET("/device/:deviceId/capture/live", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		var device model.Device
		if err := dao.Collection("device").Find(bson.M{"deviceId": c.Param("deviceId")}).One(&device); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		} else {
			service.ServeLiveStream(service.NewClientOpts(config.Get().MQTTBroker), deviceId, c)
		}
	})

	r.GET("/device/:deviceId/events", func(c *gin.Context) {
		events := make([]model.Event, 0)
		if err := dao.Collection("event").
			Find(bson.M{"deviceId": c.Param("deviceId")}).
			Sort("-timestamp").
			Limit(50).All(&events);
			err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, events)
		}
	})

	r.GET("/device/:deviceId/startRecognize", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		totalPics := 12
		if c.Query("totalPics") != "" {
			if pic, err := strconv.Atoi(c.Query("totalPics")); err == nil && pic > 0 {
				totalPics = pic
			}
		}
		frameDelay := 250
		if c.Query("frameDelay") != "" {
			if delay, err := strconv.Atoi(c.Query("frameDelay")); err == nil && delay > 0 {
				frameDelay = delay
			}
		}

		var device model.Device
		if err := dao.Collection("device").Find(bson.M{"deviceId": deviceId}).One(&device); err != nil {
			c.JSON(500, gin.H{"error": "device not exists"})
			return
		}

		if frames, err := service.CaptureFrameContinuously(service.NewClientOpts(config.Get().MQTTBroker),
			deviceId, frameDelay, totalPics); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		} else {
			if response, err := service.CallBulkRecognize(service.NewClientOpts(config.Get().MQTTBroker), device.ProjectId, frames); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			} else {
				result := make([]RecognizeResponse, 0)
				for i, fd := range response.FaceDetailsList {
					result = append(result, RecognizeResponse{
						Image:           frames[i],
						FaceDetailsList: fd,
					})
				}
				c.JSON(200, result)
			}
		}
	})
}

type RecognizeResponse struct {
	Image           []byte              `json:"image"`
	FaceDetailsList []model.FaceDetails `json:"faceDetailsList"`
}
