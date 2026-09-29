package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"masterbook/internal/httpx"
)

type contextKey string

const userIDKey contextKey = "userID"
const roleKey contextKey = "role"

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
				httpx.Error(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			token, err := jwt.ParseWithClaims(parts[1], &UserClaims{}, func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				httpx.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(*UserClaims)
			if !ok || claims.Subject == "" {
				httpx.Error(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			userID, err := strconv.ParseInt(claims.Subject, 10, 64)
			if err != nil || userID <= 0 {
				httpx.Error(w, http.StatusUnauthorized, "invalid token subject")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (int64, bool) {
	value := r.Context().Value(userIDKey)
	userID, ok := value.(int64)
	return userID, ok && userID > 0
}

func GetRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(roleKey).(string)
	return role, ok && role != ""
}

// RequireRole пропускает дальше только пользователей с одной из перечисленных
// ролей. Должен оборачивать хендлер изнутри AuthMiddleware — сначала должен
// отработать AuthMiddleware, чтобы роль уже была в контексте.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRole(r)
			if !ok || !allowed[role] {
				httpx.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
