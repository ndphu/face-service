package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"framed-mqtt-client/controller"
	"framed-mqtt-client/db"
	"framed-mqtt-client/model"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hybridgroup/mjpeg"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var streamLock = sync.Mutex{}
var streams = make(map[string]map[string]*mjpeg.Stream)
var deviceLock = sync.Mutex{}
var devices = make(map[string]bool)
var frameLock = sync.RWMutex{}
var currentFrame = make(map[string][]byte)

func main() {
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://35.197.155.112:4443"
	}

	devices["rpi-00000000ece92c87"] = true
	devices["window-59a3b558-2207-45a3-8535-b61fc6fe454f"] = true

	rawStream := mjpeg.NewStream()
	rawStream.FrameInterval = 25 * time.Millisecond

	// "tcp://35.197.155.112:4443"

	clientId := uuid.New().String()
	log.Println("Connecting to MQTT with client ID:", clientId)
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientId)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT")
		deviceLock.Lock()
		for deviceId := range devices {
			topic := "/3ml/device/" + deviceId + "/framed/out"
			log.Println("subscribed to", topic)
			c.Subscribe(topic, 0, func(client mqtt.Client, message mqtt.Message) {
				streamLock.Lock()
				for _, s := range streams[topic] {
					if s != nil {
						s.UpdateJPEG(message.Payload())
					}
				}
				streamLock.Unlock()
				frameLock.RLock()
				currentFrame[deviceId] = message.Payload()
				frameLock.RUnlock()
			}).Wait()
		}

		deviceLock.Unlock()
	}

	mqttClient := mqtt.NewClient(opts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error().Error())
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Content-Length", "X-Requested-With", "Connection", "Upgrade"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}))

	r.Handle("GET", "/api/device/:deviceId/capture/live", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
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

	r.Handle("GET", "/api/device/:deviceId/recognize/live", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		s := mjpeg.NewStream()
		go func() {
			for {
				select {
				case <-c.Request.Context().Done():
					return
				default:
					frameLock.Lock()
					frame := append([]byte{}, currentFrame[deviceId]...)
					frameLock.Unlock()
					base64Img := base64.StdEncoding.EncodeToString(frame)
					response, err := callRPC(base64Img, mqttClient)

					if err != nil {
						log.Println(err.Error())
						c.JSON(500, gin.H{"error": err.Error()})
						return
					} else if response.Error != nil {
						log.Println(err.Error())
						c.JSON(500, gin.H{"error": response.Error.Error()})
						return
					} else {
						for _, f := range response.RecognizedFaces {
							log.Println(f.Label)
						}
						s.UpdateJPEG(frame)
					}
				}
			}
		}()
		s.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/api/device/:deviceId/capture/snap", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		c.Writer.Header().Set("Content-Type", "image/jpeg")
		c.Writer.Write(currentFrame[deviceId])
	})

	r.GET("/api/device/:deviceId/detectFaces2", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		base64Img := base64.StdEncoding.EncodeToString(currentFrame[deviceId])
		requestId := uuid.New().String()

		if msg, err := json.Marshal(DetectRequest{
			RequestId: requestId,
			Payload:   base64Img,
		}); err != nil {
			panic(err)
		} else {
			rspc := make(chan DetectResponse)
			rspTopic := "/3ml/detect/response/" + requestId
			if tok := mqttClient.Subscribe(rspTopic, 0, func(client mqtt.Client, message mqtt.Message) {
				var dr DetectResponse
				if err := json.Unmarshal(message.Payload(), &dr); err != nil {
					panic(err)
				} else {
					rspc <- DetectResponse{}
				}
			}); tok.Wait() && tok.Error() != nil {
				panic(tok.Error())
			} else {
				log.Println("subscribed to response channel", rspTopic)
			}

			if tok := mqttClient.Publish("/3ml/detect/request", 0, false, msg); tok.Wait() && tok.Error() != nil {
				panic(tok.Error())
			} else {
				log.Println("sent request to request topic")
			}

			dr := <-rspc
			c.JSON(200, gin.H{
				"image":         base64Img,
				"detectedFaces": dr.DetectedFaces,
			})
		}
	})

	r.GET("/api/device/:deviceId/detectFaces", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		base64Img := base64.StdEncoding.EncodeToString(currentFrame[deviceId])

		//mqttClient.Publish("/3ml/detect/request", 0, )
		if body, err := json.Marshal(&ImageInput{
			Payload: base64Img,
		}); err != nil {
			panic(err)
		} else {
			if req, err := http.NewRequest("POST", "http://192.168.40.137:9999/api/detectFaces", bytes.NewBuffer(body));
				err != nil {
				panic(err)
			} else {
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()
				body, _ := ioutil.ReadAll(resp.Body)
				var dr DetectResponse
				if err := json.Unmarshal(body, &dr); err != nil {
					panic(err)
				} else {
					c.JSON(200, gin.H{
						"image":         base64Img,
						"detectedFaces": dr.DetectedFaces,
					})
				}
			}
		}
	})

	r.GET("/api/device/:deviceId/recognizeFaces", func(c *gin.Context) {
		deviceId := c.Param("deviceId")
		frameLock.Lock()
		base64Img := base64.StdEncoding.EncodeToString(currentFrame[deviceId])
		frameLock.Unlock()
		if resp, err := callRPC(base64Img, mqttClient); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{
				"image":           base64Img,
				"recognizedFaces": resp.RecognizedFaces,
			})
		}

	})

	controller.LabelController(r.Group("/api"))
	r.Group("/api").GET("/device/:deviceId/faceInfos", func(c *gin.Context) {
		var faces []model.Face
		if err := dao.Collection("face").Find(nil).All(&faces); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, faces)
		}
	})

	r.Group("/api").GET("/device/:deviceId/reloadSamples", func(c *gin.Context) {
		device := model.Device{
			DeviceId: c.Param("deviceId"),
		}

		if err := device.ReloadSamples(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, gin.H{})
		}
	})

	r.Run()

}

func callRPC(base64Img string, mqttClient mqtt.Client) (*RecognizedResponse, error) {
	reqId := uuid.New().String()
	rpcReqPayload, _ := json.Marshal(RecognizeRequest{
		RequestId: reqId,
		Payload:   base64Img,
	})
	rpcResponse := make(chan RecognizedResponse)
	rpcRequestTopic := "/3ml/rpc/recognizeFaces/request"
	rpcResponseTopic := "/3ml/rpc/recognizeFaces/response/" + reqId
	mqttClient.Subscribe(rpcResponseTopic, 0, func(client mqtt.Client, message mqtt.Message) {
		resp := RecognizedResponse{}
		if err := json.Unmarshal(message.Payload(), &resp); err != nil {
			log.Println("[RPC]", reqId, "fail to unmarshal response")
			rpcResponse <- RecognizedResponse{
				Error: err,
			}
		} else {
			log.Println("[RPC]", reqId, "received response")
			rpcResponse <- resp
		}
	}).Wait()
	start := time.Now()
	log.Println("[RPC]", reqId, "publishing request to", rpcRequestTopic)
	mqttClient.Publish(rpcRequestTopic, 0, false, rpcReqPayload).Wait()
	log.Println("[RPC]", reqId, "request published successfully.")
	log.Println("[RPC]", reqId, "waiting for response on", rpcResponseTopic)
	rpcTimeout := time.NewTimer(5 * time.Second)
	select {
	case resp := <-rpcResponse:
		log.Println("[RPC]", reqId, "received response after", time.Since(start))
		return &resp, nil
	case <-rpcTimeout.C:
		log.Println("[RPC]", reqId, "timeout occurred.")
		return nil, errors.New("timeout")
	}
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
}

type RecognizedResponse struct {
	RecognizedFaces []RecognizedFace `json:"recognizedFaces"`
	Error           error            `json:"error"`
}
