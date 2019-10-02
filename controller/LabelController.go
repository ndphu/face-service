package controller

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"face-service/auth"
	"face-service/db"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/ndphu/swd-commons/model"
)

func LabelController(r *gin.RouterGroup) {
	r.POST("/labels", func(c *gin.Context) {
		u := auth.CurrentUser(c)
		faces := make([]model.Face, 0)
		if err := c.ShouldBindJSON(&faces); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			if len(faces) > 0 {
				for _, face := range faces {
					face.Id = bson.NewObjectId()
					face.UserId = u.Id
					var buf bytes.Buffer
					if err := binary.Write(&buf, binary.LittleEndian, face.Descriptor); err != nil {
						panic(err)
					} else {
						face.MD5 = fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
						if err := dao.Collection("face").Insert(&face); err != nil {
							panic(err)
						}
					}
				}
			}
			c.JSON(200, gin.H{"error": ""})
		}
	})

	r.GET("/label/:label/descriptors", func(c *gin.Context) {
		faces := make([]model.Face, 0)
		if err := dao.Collection("face").Find(bson.M{
			"label": c.Param("label"),
		}).All(&faces); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			c.JSON(200, faces)
		}
	})

	r.POST("/label/:label/descriptors", func(c *gin.Context) {
		faces := make([]model.Face, 0)
		if err := c.ShouldBindJSON(&faces); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			if len(faces) > 0 {
				for _, face := range faces {
					face.Id = bson.NewObjectId()
					var buf bytes.Buffer
					if err := binary.Write(&buf, binary.LittleEndian, face.Descriptor); err != nil {
						panic(err)
					} else {
						face.MD5 = fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
						if err := dao.Collection("face").Insert(&face); err != nil {
							panic(err)
						}
					}
				}
			}
			c.JSON(200, gin.H{"error": ""})
		}
	})
}
