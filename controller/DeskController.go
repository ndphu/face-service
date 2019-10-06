package controller

import (
	"face-service/auth"
	"face-service/db"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/ndphu/swd-commons/model"
	"log"
)

func DeskController(r *gin.RouterGroup) {

	r.GET("/desks", func(c *gin.Context) {
		user := auth.CurrentUser(c)
		var desks []model.Desk
		if err := dao.Collection("desk").Find(bson.M{"owner": user.Id}).All(&desks); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, desks)
		}
	})

	r.POST("/desks", func(c *gin.Context) {
		user := auth.CurrentUser(c)
		var desk model.Desk
		if err := c.ShouldBindJSON(&desk); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			desk.Id = bson.NewObjectId()
			if desk.DeskId == "" {
				desk.DeskId = uuid.New().String()
			}
			desk.Owner = user.Id
			if err := dao.Collection("desk").Insert(&desk); err != nil {
				c.JSON(500, gin.H{"error": err})
				return
			} else {
				// notification rule
				if err := createDefaultRules(&desk); err != nil {
					log.Println("Fail to crate new desk by error:", err.Error())
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				c.JSON(201, desk)

			}
		}
	})

	r.GET("/desk/:deskId", func(c *gin.Context) {
		var desk model.Desk
		if err := dao.Collection("desk").Find(bson.M{
			"deskId": c.Param("deskId"),
			"owner":  auth.CurrentUser(c).Id,
		}).One(&desk); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, desk)
		}
	})

	r.GET("/desk/:deskId/faceInfos", func(c *gin.Context) {
		var faces []model.Face
		if err := dao.Collection("face").Find(bson.M{
			"deskId": c.Param("deskId"),
			"owner":  auth.CurrentUser(c).Id,
		}).All(&faces); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, faces)
		}
	})

	r.GET("/desk/:deskId/devices", func(c *gin.Context) {
		var devices []model.Device
		if err := dao.Collection("device").Find(bson.M{
			"deskId": c.Param("deskId"),
			"owner":  auth.CurrentUser(c).Id,
		}).All(&devices); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, devices)
		}
	})

	r.POST("/desk/:deskId/devices", func(c *gin.Context) {
		var device model.Device
		if err := c.ShouldBindJSON(&device); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			device.Id = bson.NewObjectId()
			device.DeskId = c.Param("deskId")
			if device.DeviceId == "" {
				device.DeviceId = uuid.New().String()
			}
			device.Owner = auth.CurrentUser(c).Id

			if err := dao.Collection("device").Insert(&device); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
			} else {
				c.JSON(201, device)
			}
		}
	})

	r.GET("/desk/:deskId/rules", func(c *gin.Context) {
		var rules = make([]model.Rule, 0)
		if err := dao.Collection("rule").Find(bson.M{"deskId": c.Param("deskId")}).All(&rules); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, rules)
		}
	})

	r.POST("/rule/:ruleId", func(c *gin.Context) {
		var rule model.Rule
		if err := c.ShouldBindJSON(&rule); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			if err := dao.Collection("rule").UpdateId(bson.ObjectIdHex(c.Param("ruleId")), &rule); err != nil {
				log.Println("Fail to update rule:", c.Param("ruleId"), "by error:", err.Error())
				c.JSON(500, gin.H{"error": err})
			} else {
				c.JSON(201, rule)
			}
		}
	})
}
func createDefaultRules(desk *model.Desk) error {
	return dao.Collection("rule").Insert(model.Rule{
		Id:              bson.NewObjectId(),
		DeskId:          desk.DeskId,
		IntervalMinutes: model.DefaultSittingRemindInterval,
		Type:            model.RuleTypeSittingMonitoring,
		UserId:          desk.Owner,
	}, model.Rule{
		Id:              bson.NewObjectId(),
		DeskId:          desk.DeskId,
		IntervalMinutes: model.DefaultDrinkRemindInterval,
		Type:            model.RuleTypeDrinkWaterReminder,
		UserId:          desk.Owner,
	})

}
