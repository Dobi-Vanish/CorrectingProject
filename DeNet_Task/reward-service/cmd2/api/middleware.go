package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

const (
	refreshTokenLength = 32
	accessTokenExpTime = time.Minute * 15
)

type UserData struct {
	ID                 int
	RefreshToken       string
	HashedRefreshToken string
	AccessToken        string
}

// generateTokens generates refresh and access tokens for the user.
func generateTokens(userID int, secretKey string) (*UserData, error) {
	accessToken, err := generateAccessToken(userID, secretKey)
	if err != nil {
		return nil, err
	}

	refreshToken, hashedRefreshToken, err := generateRefreshToken()

	if err != nil {
		return nil, err
	}

	return &UserData{
		ID:                 userID,
		RefreshToken:       refreshToken,
		HashedRefreshToken: hashedRefreshToken,
		AccessToken:        accessToken,
	}, nil
}

// validateRefreshToken validates refresh token for the user.
func validateRefreshToken(hashedRefreshToken, refreshToken string) error {
	decodedBytes, err := base64.StdEncoding.DecodeString(hashedRefreshToken)
	if err != nil {
		return fmt.Errorf("error during decoding hashed refresh token: %w", err)
	}

	err = bcrypt.CompareHashAndPassword(decodedBytes, []byte(refreshToken))
	if err != nil {
		return fmt.Errorf("invalid refresh token: %w", err)
	}

	return nil
}

func generateRefreshToken() (string, string, error) {
	token := make([]byte, refreshTokenLength)

	_, err := rand.Read(token)
	if err != nil {
		return "", "", fmt.Errorf("error during reading made token: %w", err)
	}

	refreshToken := base64.StdEncoding.EncodeToString(token)

	hashedRefreshToken, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("error during generatind refresh toke: %w", err)
	}

	return refreshToken, base64.StdEncoding.EncodeToString(hashedRefreshToken), nil
}

// generateAccessToken generates access tokens based on who was authenticated.
func generateAccessToken(userID int, secretKey string) (string, error) {
	expirationTime := time.Now().Add(accessTokenExpTime)

	claims := &jwt.MapClaims{
		"sub": userID,
		"exp": expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("error during signing the token: %w", err)
	}

	return tokenString, nil
}

// authTokenMiddleware auths users to get access to some pages only by having access token.
func (app *Config) authTokenMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("accessToken")
			if err != nil {
				app.errorJSON(w, err, http.StatusUnauthorized)

				return
			}

			tokenString := cookie.Value
			claims := &jwt.MapClaims{
				"sub": userIDKey,
			}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(secretKey), nil
			})

			if err != nil || !token.Valid {
				app.errorJSON(w, err, http.StatusUnauthorized)

				return
			}

			userID, ok := (*claims)["sub"].(float64)
			if !ok {
				app.errorJSON(w, err, http.StatusUnauthorized)

				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, int(userID))
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
