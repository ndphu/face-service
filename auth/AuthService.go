package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"face-service/db"
	"firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/ndphu/swd-commons/model"
	"github.com/ndphu/swd-commons/slack"
	"google.golang.org/api/option"
	"log"
	"os"
	"time"
)

type AuthService struct {
	App *firebase.App
}

var authService *AuthService

func GetAuthService() (*AuthService, error) {
	if authService == nil {

		rawKey, err := base64.StdEncoding.DecodeString(os.Getenv("FIREBASE_ADMIN_ACCOUNT"))
		if err != nil {
			log.Fatal("[FIREBASE]", "Fail to parse admin key")
		}

		opt := option.WithCredentialsJSON(rawKey)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Fatalln("[FIREBASE]", "Error initializing app", err)
		}

		log.Println("[FIREBASE]", "Firebase connected successfully")

		authService = &AuthService{
			App: app,
		}
	}
	return authService, nil
}

func CurrentUser(c *gin.Context) *User {
	val, _ := c.Get("user")
	user := val.(*User)
	return user
}

func CurrentJWT(c *gin.Context) string {
	val, exist := c.Get("jwtToken")
	if !exist {
		return ""
	}

	return val.(string)
}

func (s *AuthService) getAuthClient() (*auth.Client, error) {
	return s.App.Auth(context.Background())
}

func (s *AuthService) GetUserFromToken(jwtToken string) (*User, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("TOKEN_SECRET")), nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		_roles := claims["roles"].([]interface{})
		roles := make([]string, len(_roles))
		for i, role := range _roles {
			roles[i] = role.(string)
		}

		return &User{
			Id:    bson.ObjectIdHex(claims["user_id"].(string)),
			Email: claims["user_email"].(string),
			Roles: roles,
		}, nil
	} else {
		log.Println("[FIREBASE]", "Fail to parse token")
		return nil, err
	}
}

func (s *AuthService) CreateUserWithEmail(email string, password string, displayName string) (*User, error) {
	if c, err := dao.Collection("user").Find(bson.M{"email": email}).Count(); err != nil {
		log.Println("[DB]", "Fail to check user email exists")
		return nil, err
	} else if c > 0 {
		log.Println("[FIREBASE]", "User email", email, "is already existed")
		return nil, errors.New("USER_EMAIL_ALREADY_USED")
	}
	params := (&auth.UserToCreate{}).
		Email(email).
		EmailVerified(false).
		Password(password).
		DisplayName(displayName).
		Disabled(false)

	client, err := s.getAuthClient()
	if err != nil {
		return nil, err
	}

	u, err := client.CreateUser(context.Background(), params)
	if err != nil {
		log.Println("[FIREBASE]", "Error creating user:", err)
		return nil, err
	}

	log.Println("[FIREBASE]", "Successfully created user: ", u.Email)

	//TODO: transaction here
	user := User{
		Id:          bson.NewObjectId(),
		DisplayName: displayName,
		Email:       u.Email,
		Roles:       []string{"user"},
	}
	err = dao.Collection("user").Insert(&user)
	if err != nil {
		return nil, err
	}

	sendSlackInvitation(&user)

	return &user, err
}

func sendSlackInvitation(u *User) {
	log.Println("[SLACK]", "Sending slack invitation for user", u.Email)
	sc := model.SlackConfig{
		Id:             bson.NewObjectId(),
		UserId:         u.Id,
		SentInvitation: false,
	}

	if err := dao.Collection("slack_config").Insert(&sc); err != nil {
		log.Println("[DB]", "Fail to insert slack_config for user:", u.Email)
	} else {
		if err := slack.SendSlackInvitation(u.Email); err != nil {
			if err.Error() == "ALREADY_IN_TEAM" {
				// user invited, just lookup the user
				log.Println("[SLACK]", "User already in Slack Org, looking up existing User")
				if slackUser, err := slack.LookupUserIdByEmail(u.Email); err != nil {
					//TODO: handle error here
					log.Println("[SLACK]", "Fail to lookup user by email:", u.Email, "by error", err.Error())
				} else {
					sc.SlackUserId = slackUser.Id
					if dao.Collection("slack_config").UpdateId(sc.Id, &sc); err != nil {
						log.Println("[DB]", "Fail to update slack_config for user:", u.Email)
					} else {
						log.Println("[DB]", "Linked user:", u.Email, "with Slack user id", sc.SlackUserId)
					}
				}
			} else if err.Error() == "ALREADY_IN_TEAM_INVITED_USER" {
				if dao.Collection("slack_config").Update(bson.M{"userId": u.Id}, bson.M{"$set": bson.M{"sendInvitation": true}}); err != nil {
					log.Println("[DB]", "Fail to update slack_config for user:", u.Email)
				} else {
					log.Println("[DB]", "Updated user:", u.Email, "set sendInvitation = true")
				}
			}
		} else {
			sc.SentInvitation = true
			if err := dao.Collection("slack_config").UpdateId(sc.Id, &sc); err != nil {
				log.Println("[DB]", "Fail to update slack_config to set email sent to true for user:", u.Email)
			}
		}
	}
}

func (s *AuthService) LoginWithFirebaseToken(firebaseToken string) (*User, string, error) {
	client, err := s.App.Auth(context.Background())
	token, err := client.VerifyIDToken(context.Background(), firebaseToken)
	if err != nil {
		log.Println("[FIREBASE]", "Fail to parse token")
		return nil, "", err
	}
	user := User{}
	err = dao.Collection("user").
		Find(
			bson.M{
				"email": token.Claims["email"].(string),
			}).
		One(&user)

	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat":        now.Unix(),
		"exp":        now.AddDate(0, 0, 1).Unix(),
		"user_id":    user.Id.Hex(),
		"user_email": user.Email,
		"roles":      user.Roles,
		"provider":   "Firebase",
		"type":       "login_token",
	})
	jwtTokenString, err := jwtToken.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
	return &user, jwtTokenString, err
}

func (s *AuthService) NewServiceToken(user *User) (*ServiceToken, error) {

	tokenId := uuid.New().String()
	now := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat":        now.Unix(),
		"exp":        now.AddDate(1, 0, 0).Unix(),
		"user_id":    user.Id.Hex(),
		"user_email": user.Email,
		"type":       "service_token",
		"roles":      []string{"user", "service"},
		"token_id":   tokenId,
	})
	token, err := jwtToken.SignedString([]byte(os.Getenv("TOKEN_SECRET")))

	if err != nil {
		return nil, err
	}

	st := ServiceToken{
		Id:        bson.NewObjectId(),
		UserId:    user.Id,
		Token:     token,
		CreatedAt: now,
		TokenId:   tokenId,
	}

	err = dao.Collection("service_token").Insert(&st)

	return &st, err
}
