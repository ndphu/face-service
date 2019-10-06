package controller

import (
	"encoding/base64"
	"encoding/json"
	"face-service/auth"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"log"
	"os"
)

type RegisterInfo struct {
	DisplayName string `json:"displayName"`
	UserEmail   string `json:"email"`
	Password    string `json:"password"`
}

type LoginWithFirebase struct {
	Token string `json:"token"`
}

type UserInfo struct {
	Id            bson.ObjectId `json:"id" bson:"_id"`
	Email         string        `json:"email" bson:"email"`
	Roles         []string      `json:"roles" bson:"roles"`
	DisplayName   string        `json:"displayName" bson:"displayName"`
	NoOfAdminKeys int           `json:"noOfAdminKeys" bson:"noOfAdminKeys"`
	NoOfAccounts  int           `json:"noOfAccounts" bson:"noOfAccounts"`
}

var firebaseConfig FirebaseWebConfig

func AuthController(r *gin.RouterGroup) {
	authService, _ := auth.GetAuthService()


	if wcnf, err := base64.StdEncoding.DecodeString(os.Getenv("FIREBASE_WEB_CONFIG")); err != nil {
		log.Fatalln("Fail to parse firebase web config key FIREBASE_WEB_CONFIG", err)
	} else if err := json.Unmarshal(wcnf, &firebaseConfig); err != nil {
		log.Fatalln("Fail to unmarshal firebase web config key FIREBASE_WEB_CONFIG", err)
	}

	r.POST("/register", func(c *gin.Context) {
		ri := RegisterInfo{}
		err := c.ShouldBindJSON(&ri)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if user, err := authService.CreateUserWithEmail(ri.UserEmail, ri.Password, ri.DisplayName); err != nil {
			c.JSON(500, gin.H{"error": "Fail to create user with email. Error: " + err.Error()})
			return
		} else {
			c.JSON(200, gin.H{"user": user})
		}

	})

	r.POST("/login/firebase", func(c *gin.Context) {
		loginInfo := LoginWithFirebase{}
		if err := c.ShouldBindJSON(&loginInfo); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		} else {
			if user, jwtToken, err := authService.LoginWithFirebaseToken(loginInfo.Token); err != nil {
				c.JSON(500, gin.H{"error": "login fail by error: " + err.Error()})
				return
			} else {
				c.JSON(200, gin.H{"user": user, "jwtToken": jwtToken})
			}
		}
	})

	r.GET("/firebaseWebConfig", func(c *gin.Context) {
		c.JSON(200, firebaseConfig)
	})
}

type FirebaseWebConfig struct {
	ApiKey            string `json:"apiKey"`
	AuthDomain        string `json:"authDomain"`
	DatabaseURL       string `json:"databaseURL"`
	ProjectId         string `json:"projectId"`
	StorageBucket     string `json:"storageBucket"`
	MessagingSenderId string `json:"messagingSenderId"`
	AppId             string `json:"appId"`
}
