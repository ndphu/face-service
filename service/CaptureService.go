package service

import (
	"errors"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hybridgroup/mjpeg"
	"log"
	"time"
)

func CaptureFrameContinuously(deviceId string, frameDelay int, totalPics int) ([][]byte, error) {
	log.Println("[RECOGNIZE]", "Capturing", totalPics, "frames from device", deviceId, "with delay", frameDelay)
	clientId := uuid.New().String()
	opts := GetDefaultOps(clientId)

	var pics [][]byte

	done := make(chan bool)

	opts.OnConnect = func(c mqtt.Client) {
		topic := getFrameOutTopic(deviceId)

		secTicker := time.NewTicker(time.Duration(frameDelay) * time.Millisecond)
		c.Subscribe(topic, 0, func(client mqtt.Client, message mqtt.Message) {
			select {
			case <-secTicker.C:
				pics = append(pics, message.Payload())
				break
			default:
				break
			}
			if len(pics) == totalPics {
				client.Disconnect(0)
				done <- true
			}
		}).Wait()
	}

	captureClient := mqtt.NewClient(opts)
	if token := captureClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println("[RECOGNIZE] Fail to connect to MQTT", token.Error())
		return nil, token.Error()
	}

	defer captureClient.Disconnect(500)

	timeout := time.NewTimer((time.Duration(frameDelay*totalPics) * time.Millisecond) + 5*time.Second)

	select {
	case <-timeout.C:
		log.Println("[RECOGNIZE] Capture timeout.")
		return nil, errors.New("capture_timeout")
	case <-done:
		log.Println("[RECOGNIZE] Capture completed. Frame count:", len(pics))
		return pics, nil
	}
}

func ServeLiveStream(deviceId string, c *gin.Context) {
	log.Println("[LIVESTREAM]", "Serving live stream for device", deviceId)
	s := mjpeg.NewStream()

	clientId := "livestream_" + deviceId + "_" + uuid.New().String()
	opts := GetDefaultOps(clientId)

	timeoutDuration := 5 * time.Second
	frameTimeout := time.NewTimer(timeoutDuration)

	opts.OnConnect = func(c mqtt.Client) {
		topic := getFrameOutTopic(deviceId)
		c.Subscribe(topic, 0, func(client mqtt.Client, message mqtt.Message) {
			s.UpdateJPEG(message.Payload())
			frameTimeout.Reset(timeoutDuration)
		}).Wait()
	}

	liveStreamClient := mqtt.NewClient(opts)
	if token := liveStreamClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println("[LIVESTREAM] Fail to connect to MQTT", token.Error())
		return
	}

	defer liveStreamClient.Disconnect(100)

	go s.ServeHTTP(c.Writer, c.Request)

	select {
	case <-frameTimeout.C:
		log.Println("[LIVESTREAM]", "Timeout: frame does not send after", timeoutDuration, "seconds")
		// TODO: show timeout image
		//stream.UpdateJPEG()
		return

	case <-c.Request.Context().Done():
		log.Println("[LIVESTREAM]", "HTTP request ended.")
		return
	}
}

func getFrameOutTopic(deviceId string) string {
	return "/3ml/device/" + deviceId + "/framed/out"
}
