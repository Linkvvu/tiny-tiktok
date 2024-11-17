package jwt

import (
	"fmt"
	"net/http"
	"strconv"
	"tiktok/util"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const jwt_secret = "Linkvvu"

type TiktokClaim struct {
	jwt.RegisteredClaims
	UserId string `json:"user_id"`
}

func getToken(ctx *gin.Context) (token string) {
	token = ctx.Query("token")
	if token != "" {
		return
	}
	token = ctx.PostForm("token")
	return
}

func AuthorizationMiddleware(ctx *gin.Context) {
	token := getToken(ctx)
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status_code": 404,
			"status_msg":  "账号或密码错误，请检查您的账号密码",
		})
		ctx.Abort()
		return
	}

	claim, err := ParsingToken(token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"status_code": 404,
			"status_msg":  "账号或密码错误，请检查您的账号密码",
		})
		ctx.Abort()
		return
	}

	user_id, _ := strconv.Atoi(claim.UserId)
	ctx.Set("user_id", user_id)
	ctx.Next()
}

func NewToken(user_id string) (string, error) {
	claim := TiktokClaim{
		UserId: user_id,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tiktok",
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour * 24)},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	str, err := token.SignedString([]byte(util.JwtSecret))
	if err != nil {
		return "", err
	}
	return str, nil
}

func ParsingToken(token string) (TiktokClaim, error) {
	claims := &TiktokClaim{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, util.ErrInvalidJwtStatus
		}
		return []byte(jwt_secret), nil
	})
	if err != nil {
		return TiktokClaim{}, fmt.Errorf("%w: %w", util.ErrInvalidJwtStatus, err)
	}
	return *claims, nil
}
