package controller

import (
	"face-service/db"
	"face-service/model"
	"face-service/service"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"image"
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
			service.ServeLiveStream(deviceId, c)
		}
	})

	//r.GET("/device/:deviceId/recognizeFaces", func(c *gin.Context) {
	//	deviceId := c.Param("deviceId")
	//	var device model.Device
	//	if err := dao.Collection("device").Find(bson.M{"deviceId": deviceId}).One(&device); err != nil {
	//		c.JSON(500, gin.H{"error": "device not exists"})
	//		return
	//	}
	//
	//	start := time.Now()
	//	for {
	//		frameLock.Lock()
	//		base64Img := base64.StdEncoding.EncodeToString(currentFrame[deviceId])
	//		frameLock.Unlock()
	//		if resp, err := callRPC(device.ProjectId, base64Img); err != nil {
	//			c.JSON(500, gin.H{"error": err.Error()})
	//			return
	//		} else {
	//			if timeout, err := strconv.Atoi(c.Query("timeout")); err != nil {
	//				c.JSON(200, gin.H{
	//					"image":           base64Img,
	//					"recognizedFaces": resp.RecognizedFaces,
	//				})
	//				return
	//			} else {
	//				if len(resp.RecognizedFaces) == 0 {
	//					if int(time.Since(start).Seconds()) < timeout {
	//						continue
	//					}
	//					c.JSON(200, gin.H{
	//						"image":           base64Img,
	//						"recognizedFaces": resp.RecognizedFaces,
	//					})
	//					return
	//				} else {
	//					c.JSON(200, gin.H{
	//						"image":           base64Img,
	//						"recognizedFaces": resp.RecognizedFaces,
	//					})
	//					return
	//				}
	//			}
	//		}
	//	}
	//})

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

		if frames, err := service.CaptureFrameContinuously(deviceId, frameDelay, totalPics); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		} else {
			if response, err := service.CallBulkRecognize(device.ProjectId, frames); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			} else {
				result := make([]RecognizeResponse,0)
				for i, fd := range response.FaceDetailsList {
					result = append(result, RecognizeResponse{
						Image: frames[i],
						FaceDetailsList: fd,
					})
				}
				c.JSON(200, result)
			}
		}
	})
}


type ImageInput struct {
	Payload string `json:"payload"`
}

type DetectedFace struct {
	Rect       image.Rectangle `json:"rect"`
	Descriptor [128]float32    `json:"descriptor"`
}

type DetectResponse struct {
	DetectedFaces []DetectedFace `json:"detectedFaces"`
}

type DetectRequest struct {
	Payload   string `json:"payload"`
	RequestId string `json:"requestId"`
}

type RecognizeRequest struct {
	Payload   string `json:"payload"`
	RequestId string `json:"requestId"`
}

type RecognizedFace struct {
	Rect       image.Rectangle `json:"rect"`
	Label      string          `json:"label"`
	Classified int             `json:"category"`
	Descriptor [128]float32    `json:"descriptor"`
}

type RecognizedResponse struct {
	RecognizedFaces []RecognizedFace `json:"recognizedFaces"`
	Error           error            `json:"error"`
}

type RecognizeResponse struct {
	Image []byte `json:"image"`
	FaceDetailsList []model.FaceDetails `json:"faceDetailsList"`
}