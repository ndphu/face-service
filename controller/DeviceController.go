package controller

import (
	"face-service/auth"
	"face-service/config"
	"face-service/db"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/swd-commons/model"
	"github.com/ndphu/swd-commons/service"
	"log"
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
		user := auth.CurrentUser(c)
		deviceId := c.Param("deviceId")
		d := model.Device{}
		if err := dao.Collection("device").Find(bson.M{"deviceId": deviceId, "owner": user.Id}).One(&d); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

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
		if err := dao.Collection("device").Find(bson.M{"deviceId": deviceId, "owner": user.Id}).One(&device); err != nil {
			c.JSON(500, gin.H{"error": "device not exists"})
			return
		}

		if frames, err := service.CaptureFrameContinuously(service.NewClientOpts(config.Get().MQTTBroker), deviceId, frameDelay, totalPics); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		} else {
			if response, err := service.CallRecognizeWithRequest(service.NewClientOpts(config.Get().MQTTBroker), model.RecognizeRequest{
				IncludeFacesDetails: true,
				Images:              frames,
				TimeoutSeconds:      30,
			}); err != nil {
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
	r.DELETE("/device/:deviceId", func(c *gin.Context) {
		user := auth.CurrentUser(c)
		deviceId := c.Param("deviceId")
		d := model.Device{}
		if err := dao.Collection("device").Find(bson.M{"deviceId": deviceId, "owner": user.Id}).One(&d); err != nil {
			log.Println("No permission to delete or device not exists")
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := dao.Collection("device").RemoveId(d.Id); err != nil {
			log.Println("Fail to delete device", d.Id, "by error", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "device deleted"})
	})
}

type RecognizeResponse struct {
	Image           []byte              `json:"image"`
	FaceDetailsList []model.FaceDetails `json:"faceDetailsList"`
}
