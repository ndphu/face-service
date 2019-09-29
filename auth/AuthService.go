package auth

import (
	"context"
	"encoding/base64"
	"face-service/db"
	"firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
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
		log.Println("fail to parse token")
		return nil, err
	}
}

func (s *AuthService) CreateUserWithEmail(email string, password string, displayName string) (*User, error) {
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
		log.Printf("error creating user: %v\n", err)
		return nil, err
	}

	log.Printf("successfully created user: %s\n", u.Email)

	client.EmailVerificationLink(context.Background(), u.Email)

	user := User{
		Id:          bson.NewObjectId(),
		DisplayName: displayName,
		Email:       u.Email,
		Roles:       []string{"user"},
	}
	err = dao.Collection("user").Insert(&user)
	return &user, err
}

func (s *AuthService) LoginWithFirebaseToken(firebaseToken string) (*User, string, error) {
	client, err := s.App.Auth(context.Background())
	token, err := client.VerifyIDToken(context.Background(), firebaseToken)
	if err != nil {
		log.Println("fail to parse token")
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
