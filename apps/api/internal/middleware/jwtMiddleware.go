package middleware

import (
	"context"
	"net/http"
	"stock-forge/internal/jwt"
	"stock-forge/pkg"
	"strings"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func JWTMiddleware(tokenService *jwt.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				pkg.JsonErrorResponse(w, 401, "missing authorization token")
				return
			}

			claims, err := tokenService.ValidateToken(tokenStr)
			if err != nil {
				pkg.JsonErrorResponse(w, 401, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserClaims(r *http.Request) *jwt.UserClaims {
	claims, _ := r.Context().Value(ClaimsKey).(*jwt.UserClaims)
	return claims
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	cookie, err := r.Cookie("sf_access_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}