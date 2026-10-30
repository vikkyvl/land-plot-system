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
)

var testNow = time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)

func signTestToken(t *testing.T, method jwt.SigningMethod, expiresAt time.Time) string {
	t.Helper()

	tokenClaims := claims{
		Email: "student@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   testUserID,
			Issuer:    TokenIssuer,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	value, err := jwt.NewWithClaims(method, tokenClaims).SignedString([]byte(testSecret))
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
	userID, err := newTestVerifier().Verify(signTestToken(t, jwt.SigningMethodHS256, testNow.Add(time.Hour)))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if userID != testUserID {
		t.Fatalf("Verify() user ID = %q, want %q", userID, testUserID)
	}
}

func TestVerifierRejectsExpiredToken(t *testing.T) {
	_, err := newTestVerifier().Verify(signTestToken(t, jwt.SigningMethodHS256, testNow.Add(-time.Second)))
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Verify() error = %v, want ErrExpiredToken", err)
	}
}

func TestVerifierRejectsUnexpectedMethod(t *testing.T) {
	_, err := newTestVerifier().Verify(signTestToken(t, jwt.SigningMethodHS512, testNow.Add(time.Hour)))
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
	}
}
