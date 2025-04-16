package token_test

import (
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"reward-service/internal/token"
	"reward-service/pkg/consts"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

func setup() {
	os.Setenv("SECRET_KEY", "test_secret_key_1234567890")
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()
	setup()
	log.Println("secret key os variable is: ", os.Getenv("SECRET_KEY"))

	t.Run("should create generator with secret key", func(t *testing.T) {
		t.Parallel()

		g := token.NewTokenService()
		assert.NotNil(t, g)
		assert.Equal(t, "test_secret_key_1234567890", g.SecretKey)
	})
}

func TestGenerateAccessToken(t *testing.T) {
	t.Parallel()
	setup()

	tests := []struct {
		name     string
		userID   int
		setup    func()
		wantErr  bool
		validate func(t *testing.T, token string)
	}{
		{
			name:    "successful token generation",
			userID:  1,
			setup:   func() {},
			wantErr: false,
			validate: func(t *testing.T, token string) {
				t.Helper()
				parsed, err := jwt.Parse(token, func(_ *jwt.Token) (interface{}, error) {
					return []byte("test_secret_key_1234567890"), nil
				})

				require.NoError(t, err)
				assert.True(t, parsed.Valid)

				claims, ok := parsed.Claims.(jwt.MapClaims)
				require.True(t, ok, "claims should be of type MapClaims")

				// Исправленная проверка для sub
				sub, ok := claims["sub"].(float64)
				require.True(t, ok, "sub claim should be a float64")
				assert.InDelta(t, float64(1), sub, 0.0001)

				// Исправленная проверка для exp
				expVal, ok := claims["exp"].(float64)
				require.True(t, ok, "exp claim should be a float64")
				exp := time.Unix(int64(expVal), 0)
				assert.WithinDuration(t, exp, time.Now().Add(consts.AccessTokenExpireTime), time.Second)
			},
		},
	}

	for _, tt := range tests {
		res := tt
		t.Run(res.name, func(t *testing.T) {
			t.Parallel()

			if res.setup != nil {
				res.setup()
			}

			g := token.NewTokenService()
			tkn, err := g.GenerateAccessToken(res.userID)

			if res.wantErr {
				require.NoError(t, err)
				assert.NotEmpty(t, tkn)

				if res.validate != nil {
					res.validate(t, tkn)
				}
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	t.Parallel()
	setup()

	g := token.NewTokenService()

	t.Run("token should expire after specified time", func(t *testing.T) {
		t.Parallel()

		tkn, err := g.GenerateAccessToken(1)
		require.NoError(t, err)

		parser := jwt.Parser{}
		parsed, _, err := parser.ParseUnverified(tkn, jwt.MapClaims{})
		require.NoError(t, err)

		claims, ok := parsed.Claims.(jwt.MapClaims)
		require.True(t, ok)

		expVal, ok := claims["exp"].(float64)
		require.True(t, ok, "exp claim should be a float64")

		exp := time.Unix(int64(expVal), 0)

		expectedExp := time.Now().Add(consts.AccessTokenExpireTime)
		assert.WithinDuration(t, expectedExp, exp, 2*time.Second)
	})
}

func TestTokenValidation(t *testing.T) {
	t.Parallel()
	setup()

	g := token.NewTokenService() //nolint:varnamelen

	t.Run("modified token should be invalid", func(t *testing.T) {
		t.Parallel()

		tkn, err := g.GenerateAccessToken(1)
		require.NoError(t, err)

		tkn = tkn[:len(tkn)-2] + "xx"

		_, err = jwt.Parse(tkn, func(_ *jwt.Token) (interface{}, error) {
			return []byte(g.SecretKey), nil
		})

		assert.Error(t, err)
	})
}
