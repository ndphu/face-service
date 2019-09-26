package model

import (
	"github.com/globalsign/mgo/bson"
	"image"
)

type Face struct {
	Id         bson.ObjectId `json:"id" bson:"_id"`
	Label      string        `json:"label" bson:"label"`
	Descriptor [128]float32  `json:"descriptor" bson:"descriptor"`
	MD5        string        `json:"md5" bson:"md5"`
	DeviceId   string        `json:"deviceId" bson:"deviceId"`
	ProjectId  string        `json:"projectId" bson:"projectId"`
}

type RecognizeResponse struct {
	RecognizedFaces []RecognizedFace `json:"recognizedFaces"`
	Image           string           `json:"image"`
}

type RecognizedFace struct {
	Rect       image.Rectangle `json:"rect"`
	Label      string          `json:"label"`
	Classified int             `json:"category"`
}

type DetectResponse struct {
	DetectedFaces []DetectedFace `json:"detectedFaces"`
	Image         string         `json:"image"`
}

type DetectedFace struct {
	Rect       image.Rectangle `json:"rect"`
	Descriptor [128]float32    `json:"descriptor"`
}
