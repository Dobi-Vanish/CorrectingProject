package token

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"reward-service/pkg/errormsg"
)

type Validator struct {
	SecretKey string
}

// ValidateAccessToken validate provided access token.
func (ts *ServiceToken) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
	fmt.Println("access token received in ValidateAccessToken before parsing is: ", tokenString)

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
			return nil, fmt.Errorf("%w: %v", errormsg.ErrUnexpectedSigningMethod, t.Header["alg"])
		}

		return []byte(ts.SecretKey), nil
	})

	fmt.Println("access token received in ValidateAccessToken after parsing is: ", tokenString)

	if err != nil {
		return nil, errormsg.ErrTokenValidation
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errormsg.ErrInvalidToken
}
