package controller

import (
	"encoding/json"
	"face-service/db"
	"face-service/service"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ndphu/swd-commons/model"
	"log"
	"net/http"
	"sync"
)

var WSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var wsLock = sync.Mutex{}
var wsMap = make(map[string]*websocket.Conn)

var deviceNotifyLock = sync.Mutex{}
var deviceNotifyConnMap = make(map[string]map[string]bool)

func WSController(r *gin.RouterGroup) {

	monitorNotifications()

	r.GET("/ws", func(c *gin.Context) {
		if conn, err := WSUpgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
			log.Println("[WS] Failed to set WebSocket upgrade: ", err)
		} else {
			wsId := uuid.New().String()
			log.Println("[WS]", "Registering WS connection:", wsId)
			wsLock.Lock()
			wsMap[wsId] = conn
			wsLock.Unlock()

			conn.WriteJSON(WSMessage{
				Code:    200,
				Type:    "CONNECTED",
				Payload: wsId,
			})

			conn.SetCloseHandler(func(code int, text string) error {
				log.Println("[WS]", "Websocket closed code:", code, "text:", text)
				wsLock.Lock()
				delete(wsMap, wsId)
				wsLock.Unlock()
				return nil
			})

			go serveWebSocket(wsId)
		}
	})
}

func serveWebSocket(wsId string) {

	defer func() {
		log.Println("[WS]", "Stopped serving connection", wsId)
	}()

	for {
		wsLock.Lock()
		conn, exists := wsMap[wsId]
		wsLock.Unlock()
		if !exists {
			return
		}
		wsmsg := WSMessage{}
		if err := conn.ReadJSON(&wsmsg); err != nil {
			log.Println("[WS]", "Fail to read WS message for connection id", wsId, "error:", err.Error())
			return
		}
		log.Println("[WS]", "Received message:", wsmsg.Type)
		switch wsmsg.Type {
		case "WATCH_PROJECT":
			projectId := wsmsg.Payload

			if count, _ := dao.Collection("project").Find(bson.M{"projectId": projectId}).Count(); count <= 0 {
				if err := conn.WriteJSON(WSMessage{
					Code:    200,
					Type:    "APP_NOTIFICATION_WATCH_PROJECT_FAIL",
					Payload: "Project not found",
				}); err != nil {
					log.Println("[WS]", "Fail to notify watching status", wsId, "error", err.Error())
				} else {
					log.Println("[WS]", "Push notification successfully")
				}
				break
			}
			log.Println("[WS]", "Connection", wsId, "start watching project", projectId)
			deviceNotifyLock.Lock()
			if len(deviceNotifyConnMap[wsmsg.Payload]) == 0 {
				deviceNotifyConnMap[wsmsg.Payload] = make(map[string]bool)
			}
			deviceNotifyConnMap[wsmsg.Payload][wsId] = true
			deviceNotifyLock.Unlock()
			if err := conn.WriteJSON(WSMessage{
				Code:    200,
				Type:    "APP_NOTIFICATION_WATCH_PROJECT_SUCCESS",
				Payload: "You will receive notification on this project",
			}); err != nil {
				log.Println("[WS]", "Fail to notify watching status", wsId, "error", err.Error())
			} else {
				log.Println("[WS]", "Push notification successfully")
			}
			break

		case "UNWATCH_PROJECT":
			log.Println("[WS]", "Connection", wsId, "stop watching project", wsmsg.Payload)
			deviceNotifyLock.Lock()
			if len(deviceNotifyConnMap[wsmsg.Payload]) == 0 {
				deviceNotifyConnMap[wsmsg.Payload] = make(map[string]bool)
			}
			delete(deviceNotifyConnMap[wsmsg.Payload], wsId)
			deviceNotifyLock.Unlock()
			break

		}
	}
}

func monitorNotifications() {
	projects := make([]model.Project, 0)
	if err := dao.Collection("project").Find(nil).All(&projects); err != nil {
		panic(err)
	}
	clientId := uuid.New().String()
	ops := service.GetDefaultOps(clientId)
	ops.OnConnect = func(c mqtt.Client) {
		for _, p := range projects {
			c.Subscribe("/3ml/project/"+p.ProjectId+"/notification", 0, func(client mqtt.Client, message mqtt.Message) {
				log.Println("[WS]", "Notification received")
				var nf model.Notification
				if err := json.Unmarshal(message.Payload(), &nf); err != nil {
					log.Println("[WS]", "Fail to unmarshall notification", string(message.Payload()))
					return
				}
				log.Println("[WS]", "Pushing notification for project", nf.ProjectId)
				log.Println("[WS]", "Number for subscriber", len(deviceNotifyConnMap[nf.ProjectId]))
				for wsId := range deviceNotifyConnMap[nf.ProjectId] {
					wsLock.Lock()
					conn, exists := wsMap[wsId]
					wsLock.Unlock()
					if !exists {
						continue
					}

					if err := conn.WriteJSON(WSMessage{
						Code:    200,
						Type:    "APP_NOTIFICATION_REMIND",
						Payload: "You are sitting for too long. To protect you health, please consider to take a break for better health.",
					}); err != nil {
						log.Println("[WS]", "Fail to send notification of project", nf.ProjectId, "and connection", wsId, "error", err.Error())
					} else {
						log.Println("[WS]", "Push notification successfully")
					}
				}
			}).Wait()
		}
	}
	monitorClient := mqtt.NewClient(ops)

	if tok := monitorClient.Connect(); tok.Wait() && tok.Error() != nil {
		panic(tok.Error())
	}
}

type WSMessage struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}
