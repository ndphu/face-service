package service

import (
	"encoding/json"
	"errors"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/ndphu/swd-commons/model"
	"log"
	"time"
)

func CallBulkRecognize(projectId string, frames [][]byte) (*model.BulkRecognizeResponse, error) {
	reqId := uuid.New().String()
	rpcRequestTopic := "/3ml/worker/" + projectId + "/rpc/recognizeFacesBulk/request"
	rpcResponseTopic := "/3ml/worker/response/generated/" + uuid.New().String()

	rpcReqPayload, _ := json.Marshal(model.BulkRecognizeRequest{
		RequestId:           reqId,
		Images:              frames,
		IncludeFacesDetails: true,
		ResponseTo:          rpcResponseTopic,
	})

	clientId := uuid.New().String()
	opts := GetDefaultOps(clientId)
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error().Error())
	}
	defer c.Disconnect(500)

	rpcResponse := make(chan model.BulkRecognizeResponse)

	c.Subscribe(rpcResponseTopic, 0, func(c mqtt.Client, m mqtt.Message) {
		resp := model.BulkRecognizeResponse{}
		if err := json.Unmarshal(m.Payload(), &resp); err != nil {
			log.Println("[RPC]", reqId, "fail to unmarshal response")
			rpcResponse <- model.BulkRecognizeResponse{
				Error: err,
			}
		} else {
			log.Println("[RPC]", reqId, "received response")
			rpcResponse <- resp
		}
	}).Wait()

	c.Publish(rpcRequestTopic, 0, false, rpcReqPayload).Wait()

	rpcTimeout := time.NewTimer(15 * time.Second)
	select {
	case resp := <-rpcResponse:
		return &resp, nil
	case <-rpcTimeout.C:
		log.Println("[RPC]", reqId, "timeout occurred.")
		return nil, errors.New("timeout")
	}

}
