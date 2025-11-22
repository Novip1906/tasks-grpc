package utils

import (
	"fmt"
	"math/rand"
	"net/mail"
	"time"
	"unicode/utf8"

	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserId   int64  `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func EncodeJWTToken(userId int64, email, username, JWTSecretKey string) (string, error) {
	claims := TokenClaims{
		UserId:   userId,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecretKey))
}

func DecodeJWTToken(tokenString, JWTSecretKey string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func GenerateVerificationCode() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func UsernameIsValid(username string, cfg *config.Config) bool {
	length := utf8.RuneCountInString(username)
	return length >= cfg.Params.Username.Min && length <= cfg.Params.Username.Max
}

func PasswordIsValid(pass string, cfg *config.Config) bool {
	length := utf8.RuneCountInString(pass)
	return length >= cfg.Params.Password.Min && length <= cfg.Params.Password.Max
}

func EmailIsValid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
