package model

import "github.com/globalsign/mgo/bson"

type Label struct {
	Id    bson.ObjectId `json:"id" bson:"_id"`
	Label string        `json:"label"`
}
