package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret = "test_secret_that_contains_at_least_32_chars"
	testUserID = "user-1"
	testEmail  = "student@example.com"
)

var testNow = time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)

func signTestToken(t *testing.T, method jwt.SigningMethod, secret string, expiresAt time.Time) string {
	t.Helper()

	tokenClaims := claims{
		Email: testEmail,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   testUserID,
			Issuer:    TokenIssuer,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	value, err := jwt.NewWithClaims(method, tokenClaims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return value
}

func newTestVerifier() *JWTVerifier {
	verifier := NewJWTVerifier(testSecret)
	verifier.now = func() time.Time { return testNow }

	return verifier
}

func TestVerifierAcceptsValidToken(t *testing.T) {
	userID, email, err := newTestVerifier().Verify(signTestToken(t, jwt.SigningMethodHS256, testSecret, testNow.Add(time.Hour)))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if userID != testUserID || email != testEmail {
		t.Fatalf("Verify() = %q, %q", userID, email)
	}
}

func TestVerifierRejectsExpiredToken(t *testing.T) {
	_, _, err := newTestVerifier().Verify(signTestToken(t, jwt.SigningMethodHS256, testSecret, testNow.Add(-time.Second)))
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Verify() error = %v, want ErrExpiredToken", err)
	}
}

func TestVerifierRejectsInvalidTokens(t *testing.T) {
	tests := map[string]string{
		"wrong signature": signTestToken(
			t,
			jwt.SigningMethodHS256,
			"different_secret_that_contains_32_chars",
			testNow.Add(time.Hour),
		),
		"unexpected method": signTestToken(t, jwt.SigningMethodHS512, testSecret, testNow.Add(time.Hour)),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, err := newTestVerifier().Verify(token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}
