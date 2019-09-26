package model

import "github.com/globalsign/mgo/bson"

type Project struct {
	Id        bson.ObjectId `json:"id" bson:"_id"`
	ProjectId string        `json:"projectId" bson:"projectId"`
	Name      string        `json:"name" bson:"name"`
}
