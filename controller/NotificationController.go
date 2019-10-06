package controller

import (
	"face-service/auth"
	"face-service/db"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/swd-commons/model"
	"github.com/ndphu/swd-commons/slack"
	"log"
)

func NotificationController(r *gin.RouterGroup) {
	r.GET("/slackConfig", func(c *gin.Context) {
		user := auth.CurrentUser(c)
		sc := model.SlackConfig{}
		if err := dao.Collection("slack_config").Find(bson.M{"userId": user.Id}).One(&sc); err != nil {
			c.JSON(200, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, sc)
		}
	})

	r.GET("/sendSlackInvitation", func(c *gin.Context) {
		user := auth.CurrentUser(c)
		log.Println("[SLACK]", "Requesting Slack invitation for user", user.Email)
		if err := slack.SendSlackInvitation(user.Email); err != nil {
			switch err.Error() {
			case "ALREADY_IN_TEAM":
				log.Println("[SLACK]", "User with email", user.Email, "is already in Slack team. Linking user email and Slack user")
				if slackUser, err := slack.LookupUserIdByEmail(user.Email); err != nil {
					log.Println("[SLACK]", "Fail to lookup user by email:", user.Email, "by error", err.Error())
					c.JSON(500, gin.H{"error": err.Error()})
					return
				} else {
					if dao.Collection("slack_config").Update(bson.M{"userId": user.Id},bson.M{"$set": bson.M{"slackUserId": slackUser.Id}}); err != nil {
						log.Println("[DB]", "Fail to update slack_config for user:", user.Email)
						c.JSON(500, gin.H{"error": err.Error()})
					} else {
						log.Println("[DB]", "Linked user:", user.Email, "with Slack user id", slackUser.Id)
						c.JSON(200, gin.H{"error": ""})
					}
				}
				break
			case "ALREADY_IN_TEAM_INVITED_USER":
				c.JSON(500, gin.H{"error": "Email already sent"})
				return
			default:
				log.Println("[SLACK]", "Fail to send user email invitation", err.Error())
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
	})

	r.GET("/testSlackNotification", func(c *gin.Context) {
		user := auth.CurrentUser(c)
		sc := model.SlackConfig{}
		if err := dao.Collection("slack_config").Find(bson.M{"userId": user.Id}).One(&sc); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			if sc.UserId == "" {
				if slackUser, err := slack.LookupUserIdByEmail(user.Email); err != nil {
					log.Println("[SLACK]", "Fail to lookup user in Slack Org. User may not confirm the invitation")
					c.JSON(500, gin.H{"error": "User not found in Slack Organization"})
					return
				} else {
					log.Println("[SLACK]", "Found Slack user id", slackUser.Id, "for email", user.Email)
					sc.SlackUserId = slackUser.Id
					if err := dao.Collection("slack_config").UpdateId(sc.Id, sc); err != nil {
						c.JSON(500, gin.H{"error": "Fail to update user to DB"})
						return
					}
				}
			}
			if err := slack.SendSimpleTextMessageToUser(sc.SlackUserId, "This is a test notification."); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
			} else {
				c.JSON(200, gin.H{})
			}
		}
	})

	r.POST("/notificationConfig", func(c *gin.Context) {

	})
}
