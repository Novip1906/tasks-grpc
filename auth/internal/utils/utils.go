package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func EncodeJWTToken(username, JWTSecretKey string) (string, error) {
	payload := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	return token.SignedString([]byte(JWTSecretKey))
}

func DecodeJWTToken(tokenString, JWTSecretKey string) (username string, exp int64, err error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecretKey), nil
	})
	if err != nil {
		return "", 0, err
	}
	if !token.Valid {
		return "", 0, errors.New("invalid token")
	}

	if sub, ok := claims["sub"].(string); ok {
		username = sub
	} else {
		return "", 0, errors.New("username not found in token")
	}

	if expVal, ok := claims["exp"].(float64); ok {
		exp = int64(expVal)
	} else {
		return "", 0, errors.New("exp not found in token")
	}

	return username, exp, nil
}
