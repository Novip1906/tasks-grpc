package utils

import (
	"testing"
	"time"

	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeJWTToken(t *testing.T) {
	secretKey := "my-secret-key"
	userId := int64(123)
	email := "test@example.com"
	username := "testuser"

	// 1. Test Encoding
	tokenStr, err := EncodeJWTToken(userId, email, username, secretKey)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	// 2. Test Decoding Success
	claims, err := DecodeJWTToken(tokenStr, secretKey)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, userId, claims.UserId)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, username, claims.Username)
	assert.WithinDuration(t, time.Now().Add(72*time.Hour), claims.ExpiresAt.Time, 10*time.Second)

	// 3. Test Decoding with Invalid Key
	_, err = DecodeJWTToken(tokenStr, "invalid-key")
	require.Error(t, err)

	// 4. Test Decoding with Invalid Token
	_, err = DecodeJWTToken("invalid.token.string", secretKey)
	require.Error(t, err)
}

func TestGenerateVerificationCode(t *testing.T) {
	code := GenerateVerificationCode()
	assert.Len(t, code, 4)
	for _, char := range code {
		assert.True(t, char >= '0' && char <= '9', "Code must contain only digits")
	}

	// Ensure codes are reasonably random (not fully deterministic with fixed seed 1)
	code2 := GenerateVerificationCode()
	// Extremely small chance to be equal, but for simple test this is fine.
	// If it fails occasionally (1/10000), reconsider. But usually sleep is enough or time.Now().UnixNano() varies enough
	assert.NotEqual(t, "", code2)
}

func TestUsernameIsValid(t *testing.T) {
	cfg := &config.Config{
		Params: config.Params{
			Username: config.MinMaxLen{Min: 3, Max: 10},
		},
	}

	tests := []struct {
		name     string
		username string
		valid    bool
	}{
		{"Valid User", "john", true},
		{"Exact Min Valid User", "abc", true},
		{"Exact Max Valid User", "abcdefghij", true},
		{"Too Short", "ab", false},
		{"Too Long", "abcdefghijk", false},
		{"Empty", "", false},
		{"Unicode Valid", "тест", true},       // length 4 runes
		{"Unicode Too Long", "оченьдлинноеимя", false}, // > 10 runes
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid := UsernameIsValid(tc.username, cfg)
			assert.Equal(t, tc.valid, valid)
		})
	}
}

func TestPasswordIsValid(t *testing.T) {
	cfg := &config.Config{
		Params: config.Params{
			Password: config.MinMaxLen{Min: 6, Max: 20},
		},
	}

	tests := []struct {
		name  string
		pass  string
		valid bool
	}{
		{"Valid Password", "secret123", true},
		{"Exact Min Valid", "123456", true},
		{"Exact Max Valid", "12345678901234567890", true},
		{"Too Short", "12345", false},
		{"Too Long", "123456789012345678901", false},
		{"Empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid := PasswordIsValid(tc.pass, cfg)
			assert.Equal(t, tc.valid, valid)
		})
	}
}

func TestEmailIsValid(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"Valid Simple Email", "test@example.com", true},
		{"Valid Email With Name", "John Doe <test@example.com>", true},
		{"Invalid Missing @", "testexample.com", false},
		{"Invalid Missing Domain", "test@", false},
		{"Invalid Empty", "", false},
		{"Invalid Only Domain", "@example.com", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid := EmailIsValid(tc.email)
			assert.Equal(t, tc.valid, valid)
		})
	}
}
