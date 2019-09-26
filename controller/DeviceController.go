package controller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"face-service/config"
	"face-service/db"
	"face-service/model"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/hybridgroup/mjpeg"
	"image"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var streamLock = sync.Mutex{}
var streams = make(map[string]map[string]*mjpeg.Stream)
var deviceLock = sync.Mutex{}
var devices = make(map[string]bool)
var frameLock = sync.RWMutex{}
var currentFrame = make(map[string][]byte)
var topicDeviceRegex = regexp.MustCompile(`^/3ml/device/(.*)/framed/out$`)

func DeviceController(r *gin.RouterGroup) {
	clientId := uuid.New().String()
	log.Println("Connecting to MQTT with client ID:", clientId)
	opts := mqtt.NewClientOptions().AddBroker(config.Get().MQTTBroker).SetClientID(clientId)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT")
		deviceLock.Lock()
		for deviceId := range devices {
			topic := getFrameOutTopic(deviceId)
			log.Println("Subscribed to", topic)
			c.Subscribe(topic, 0, func(client mqtt.Client, message mqtt.Message) {
				handleFrame(message)
			}).Wait()
		}

		deviceLock.Unlock()
	}

	opts.OnConnectionLost = func(c mqtt.Client, e error) {
		log.Println("Loss MQTT connection by error:", e.Error())
	}

	mqttClient := mqtt.NewClient(opts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error().Error())
	}

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
		deviceLock.Lock()

		if _, exists := devices[deviceId]; !exists {
			log.Println("Register device", deviceId)
			devices[deviceId] = true
			mqttClient.Subscribe(getFrameOutTopic(deviceId), 0, func(c mqtt.Client, m mqtt.Message) {
				handleFrame(m)
			}).Wait()
		}

		deviceLock.Unlock()

		topic := "/3ml/device/" + deviceId + "/framed/out"
		sessionId := uuid.New().String()
		s := mjpeg.NewStream()
		streamLock.Lock()
		if streams[topic] == nil {
			streams[topic] = make(map[string]*mjpeg.Stream)
		}
		streams[topic][sessionId] = s

		streamLock.Unlock()

		defer func() {
			log.Println("release stream")
			streams[topic][sessionId] = nil
		}()
		s.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/device/:deviceId/recognizeFaces", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		start := time.Now()
		for {
			frameLock.Lock()
			base64Img := base64.StdEncoding.EncodeToString(currentFrame[deviceId])
			frameLock.Unlock()
			if resp, err := callRPC(base64Img); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			} else {
				if timeout, err := strconv.Atoi(c.Query("timeout")); err != nil {
					c.JSON(200, gin.H{
						"image":           base64Img,
						"recognizedFaces": resp.RecognizedFaces,
					})
					return
				} else {
					if len(resp.RecognizedFaces) == 0 {
						if int(time.Since(start).Seconds()) < timeout {
							continue
						}
						c.JSON(200, gin.H{
							"image":           base64Img,
							"recognizedFaces": resp.RecognizedFaces,
						})
						return
					} else {
						c.JSON(200, gin.H{
							"image":           base64Img,
							"recognizedFaces": resp.RecognizedFaces,
						})
						return
					}
				}
			}
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
}

func callRPC(base64Img string) (*RecognizedResponse, error) {
	reqId := uuid.New().String()
	rpcReqPayload, _ := json.Marshal(RecognizeRequest{
		RequestId: reqId,
		Payload:   base64Img,
	})
	rpcResponse := make(chan RecognizedResponse)
	rpcRequestTopic := "/3ml/rpc/recognizeFaces/request"
	rpcResponseTopic := "/3ml/rpc/recognizeFaces/response/" + reqId

	clientId := uuid.New().String()

	opts := GetDefaultOps(clientId)

	c := mqtt.NewClient(opts)

	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error().Error())
	}

	defer c.Disconnect(500)

	c.Subscribe(rpcResponseTopic, 0, func(c mqtt.Client, m mqtt.Message) {
		resp := RecognizedResponse{}
		if err := json.Unmarshal(m.Payload(), &resp); err != nil {
			log.Println("[RPC]", reqId, "fail to unmarshal response")
			rpcResponse <- RecognizedResponse{
				Error: err,
			}
		} else {
			log.Println("[RPC]", reqId, "received response")
			rpcResponse <- resp
		}
	}).Wait()
	c.Publish(rpcRequestTopic, 9, false, rpcReqPayload).Wait()
	rpcTimeout := time.NewTimer(5 * time.Second)
	select {
	case resp := <-rpcResponse:
		return &resp, nil
	case <-rpcTimeout.C:
		log.Println("[RPC]", reqId, "timeout occurred.")
		return nil, errors.New("timeout")
	}
}

func GetDefaultOps(clientId string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions().AddBroker(config.Get().MQTTBroker).SetClientID(clientId)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)
	return opts
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
