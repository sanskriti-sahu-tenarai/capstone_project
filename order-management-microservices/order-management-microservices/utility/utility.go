package utility

import (
	"time"

	"order-management/middleware"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(username, role string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(middleware.SecretKey)
}
