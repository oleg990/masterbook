package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func CreateToken(
	userID int64,
	email string,
	role string,
	secret string,
) (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		Subject:   formatUserID(userID),
		ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	token.Claims = &UserClaims{
		RegisteredClaims: claims,
		Email:            email,
		Role:             role,
	}

	return token.SignedString([]byte(secret))
}

type UserClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}

func formatUserID(id int64) string {
	return fmt.Sprintf("%d", id)
}
