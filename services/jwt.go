package services

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/CPU-commits/Intranet_BClassroom/settings"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v4"
)

var jwtKey = settings.GetSettings().JWT_SECRET_KEY

type Claims struct {
	ID       string
	UserType string
	Name     string
}

func extractToken(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	strArr := strings.Split(bearerToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func VerifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := extractToken(r)
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtKey), nil
	})
	if !token.Valid {
		return nil, errors.New("Unauthorized")
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

func ExtractTokenMetadata(token *jwt.Token) (*Claims, error) {
	claim := token.Claims.(jwt.MapClaims)
	return &Claims{
		ID:       fmt.Sprintf("%v", claim["_id"]),
		UserType: fmt.Sprintf("%v", claim["user_type"]),
		Name:     fmt.Sprintf("%v", claim["name"]),
	}, nil
}

func NewClaimsFromContext(ctx *gin.Context) (*Claims, bool) {
	user, exists := ctx.Get("user")
	if !exists {
		return &Claims{}, false
	}
	return &Claims{
		ID:       user.(*Claims).ID,
		UserType: user.(*Claims).UserType,
		Name:     user.(*Claims).Name,
	}, true
}
