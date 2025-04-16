package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"reward-service/internal/token"
)

// Auth middleware checks JWT token from cookies.
func Auth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accessCookie, err := r.Cookie("accessToken")
			if err != nil {
				handleAuthError(w, "missing access token")

				return
			}

			tokenService := token.NewTokenService()

			claims, err := tokenService.ValidateAccessToken(accessCookie.Value)
			if err != nil {
				handleAuthError(w, "invalid access token: "+err.Error())

				return
			}

			userID, ok := claims["sub"].(float64)
			if !ok {
				handleAuthError(w, "invalid user ID in token")

				return
			}

			ctx := context.WithValue(r.Context(), "userID", int(userID)) //nolint: revive, staticcheck
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleAuthError handle errors from Auth middleware.
func handleAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   true,
		"message": "Authentication failed: " + message,
	})

	if err != nil {
		return
	}
}
