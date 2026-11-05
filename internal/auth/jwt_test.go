package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseToken_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_secret_key_123")

	username := "admin"
	tokenStr, err := GenerateToken(username)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	parsedToken, err := ParseToken(tokenStr)
	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(*Claims)
	require.True(t, ok)
	assert.Equal(t, username, claims.Username)
}

func TestParseToken_InvalidSignature(t *testing.T) {
	t.Setenv("JWT_SECRET", "key_A")
	tokenStr, err := GenerateToken("admin")
	require.NoError(t, err)

	// Change secret key
	t.Setenv("JWT_SECRET", "key_B")

	_, err = ParseToken(tokenStr)
	assert.Error(t, err)
}

func TestParseToken_MalformedToken(t *testing.T) {
	_, err := ParseToken("not.a.valid.jwt")
	assert.Error(t, err)
}

func TestParseToken_Expired(t *testing.T) {
	claims := Claims{
		Username: "expired_user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("super_secret_key"))
	require.NoError(t, err)

	_, err = ParseToken(tokenStr)
	assert.Error(t, err)
}
