package service

import (
	"face-service/config"
	"github.com/eclipse/paho.mqtt.golang"
	"time"
)

func GetDefaultOps(clientId string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions().AddBroker(config.Get().MQTTBroker).SetClientID(clientId)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)
	return opts
}
