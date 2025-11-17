package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func EncodeJWTToken(userId int64, JWTSecretKey string) (string, error) {
	payload := jwt.MapClaims{
		"sub": userId,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return token.SignedString([]byte(JWTSecretKey))
}

func DecodeJWTToken(tokenString, JWTSecretKey string) (userId int64, exp int64, err error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecretKey), nil
	})
	if err != nil {
		return 0, 0, err
	}
	if !token.Valid {
		return 0, 0, errors.New("invalid token")
	}

	if sub, ok := claims["sub"].(float64); ok {
		userId = int64(sub)
	} else {
		return 0, 0, errors.New("id not found in token")
	}

	if expVal, ok := claims["exp"].(float64); ok {
		exp = int64(expVal)
	} else {
		return 0, 0, errors.New("exp not found in token")
	}

	return userId, exp, nil
}

func GenerateVerificationCode() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func UsernameIsValid(username string, cfg *config.Config) bool {
	return len(username) >= cfg.Params.Username.Min && len(username) <= cfg.Params.Username.Max
}

func PasswordIsValid(pass string, cfg *config.Config) bool {
	return len(pass) >= cfg.Params.Password.Min && len(pass) <= cfg.Params.Password.Max
}

func EmailIsValid(email string, cfg *config.Config) bool {
	return strings.Contains(email, "@")
}
