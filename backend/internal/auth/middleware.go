package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "authorization header is required",
				})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)

			if len(parts) != 2 || parts[0] != "Bearer" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid authorization header",
				})
				return
			}

			tokenString := parts[1]

			token, err := jwt.ParseWithClaims(
				tokenString,
				&UserClaims{},
				func(token *jwt.Token) (any, error) {
					if token.Method != jwt.SigningMethodHS256 {
						return nil, jwt.ErrSignatureInvalid
					}

					return []byte(secret), nil
				},
			)

			if err != nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid or expired token",
				})
				return
			}

			claims, ok := token.Claims.(*UserClaims)

			if !ok || claims.Subject == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid token claims",
				})
				return
			}

			ctx := context.WithValue(
				r.Context(),
				userIDKey,
				claims.Subject,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (string, bool) {
	value := r.Context().Value(userIDKey)

	userID, ok := value.(string)

	return userID, ok
}
