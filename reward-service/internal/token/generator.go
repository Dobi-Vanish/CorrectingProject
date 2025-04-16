package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt"
	"os"
	"reward-service/pkg/consts"
	"time"
)

type ServiceToken struct {
	SecretKey string
}

func NewTokenService() *ServiceToken {
	return &ServiceToken{
		SecretKey: os.Getenv("SECRET_KEY"),
	}
}

// GenerateTokens when called generates access tokens.
func (ts *ServiceToken) GenerateTokens(userID int) (string, string, error) {
	accessToken, err := ts.GenerateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	hashedRefreshToken, err := GenerateRefreshToken()

	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if err != nil {
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, hashedRefreshToken, nil
}

// GenerateAccessToken generates access tokens.
func (ts *ServiceToken) GenerateAccessToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(consts.AccessTokenExpireTime).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	signedToken, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))
	if err != nil {
		return "", fmt.Errorf("failed to sign the token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken generates refresh tokens.
func GenerateRefreshToken() (string, error) {
	tokenBytes := make([]byte, consts.RefreshTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	refreshToken := base64.URLEncoding.EncodeToString(tokenBytes)

	return refreshToken, nil
}
