package controller

import (
	"face-service/db"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/ndphu/swd-commons/model"
)

func ProjectController(r *gin.RouterGroup) {

	r.GET("/projects", func(c *gin.Context) {
		var projects []model.Project

		if err := dao.Collection("project").Find(nil).All(&projects); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, projects)
		}
	})

	r.POST("/projects", func(c *gin.Context) {
		var project model.Project
		if err := c.ShouldBindJSON(&project); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			project.Id = bson.NewObjectId()
			if project.ProjectId == "" {
				project.ProjectId = uuid.New().String()
			}
			if err := dao.Collection("project").Insert(&project); err != nil {
				c.JSON(500, gin.H{"error": err})
			} else {
				c.JSON(201, project)
			}
		}
	})

	r.GET("/project/:projectId", func(c *gin.Context) {
		var project model.Project
		if err := dao.Collection("project").Find(bson.M{"projectId": c.Param("projectId")}).One(&project); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, project)
		}
	})

	r.GET("/project/:projectId/faceInfos", func(c *gin.Context) {
		var faces []model.Face
		if err := dao.Collection("face").Find(bson.M{"projectId": c.Param("projectId")}).All(&faces); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, faces)
		}
	})

	r.GET("/project/:projectId/devices", func(c *gin.Context) {
		var devices []model.Device
		if err := dao.Collection("device").Find(bson.M{"projectId": c.Param("projectId")}).All(&devices); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, devices)
		}
	})

	r.POST("/project/:projectId/devices", func(c *gin.Context) {
		var device model.Device
		if err := c.ShouldBindJSON(&device); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			device.Id = bson.NewObjectId()
			device.ProjectId = c.Param("projectId")

			if device.DeviceId == "" {
				device.DeviceId = uuid.New().String()
			}
			if err := dao.Collection("device").Insert(&device); err != nil {
				c.JSON(500, gin.H{"error": err})
			} else {
				c.JSON(201, device)
			}
		}
	})

	r.GET("/project/:projectId/rules", func(c *gin.Context) {
		var rules = make([]model.Rule, 0)
		if err := dao.Collection("rule").Find(bson.M{"projectId": c.Param("projectId")}).All(&rules); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, rules)
		}
	})

	r.POST("/project/:projectId/rules", func(c *gin.Context) {
		var rule model.Rule
		if err := c.ShouldBindJSON(&rule); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			rule.Id = bson.NewObjectId()
			rule.ProjectId = c.Param("projectId")

			if err := dao.Collection("rule").Insert(&rule); err != nil {
				c.JSON(500, gin.H{"error": err})
			} else {
				c.JSON(201, rule)
			}
		}
	})
}

//func handleFrame(message mqtt.Message) {
//	streamLock.Lock()
//	for _, s := range streams[message.Topic()] {
//		if s != nil {
//			s.UpdateJPEG(message.Payload())
//		}
//	}
//	streamLock.Unlock()
//	frameLock.Lock()
//	deviceId := topicDeviceRegex.FindStringSubmatch(message.Topic())[1]
//	currentFrame[deviceId] = message.Payload()
//	frameLock.Unlock()
//}
//
//func syncFacesData(deviceId string) error {
//	clientId := uuid.New().String()
//	opts := mqtt.NewClientOptions().AddBroker(config.Get().MQTTBroker).SetClientID(clientId)
//	opts.SetKeepAlive(2 * time.Second)
//	opts.SetPingTimeout(1 * time.Second)
//	opts.SetConnectTimeout(30 * time.Second)
//
//	mqttClient := mqtt.NewClient(opts)
//
//	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
//		return token.Error()
//	}
//
//	defer mqttClient.Disconnect(500)
//
//	token := mqttClient.Publish("/3ml/rpc/sync/request", 0, false, "")
//	token.Wait()
//	return token.Error()
//}
