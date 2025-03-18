package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"tiktok/config"
	"tiktok/pkg"
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

func AuthorizationHandler(ctx *gin.Context) {
	token := getToken(ctx)
	var appE *pkg.AppError
	if token == "" {
		appE = pkg.NewError(pkg.ErrValidation, fmt.Errorf("login first please!"))
		ctx.AbortWithError(appE.HttpStatus, appE)
		return
	}

	claim, err := ParsingToken(token)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			appE = pkg.NewError(pkg.ErrAuthException, err)
		} else {
			appE = pkg.NewError(pkg.ErrUnmatchedPwd, err)
		}
		ctx.AbortWithError(appE.HttpStatus, appE)
		return
	}

	user_id, _ := strconv.ParseUint(claim.UserId, 10, 64)
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
	str, err := token.SignedString([]byte(config.JwtSecret))
	if err != nil {
		return "", err
	}
	return str, nil
}

func ParsingToken(token string) (TiktokClaim, error) {
	claims := &TiktokClaim{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwt_secret), nil
	})
	if err != nil {
		return TiktokClaim{}, err
	}
	return *claims, nil
}
