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

func newTestIssuer() *JWTIssuer {
	issuer := NewJWTIssuer(testSecret, time.Hour)
	issuer.now = func() time.Time { return testNow }

	return issuer
}

func newTestVerifier(now time.Time) *JWTVerifier {
	verifier := NewJWTVerifier(testSecret)
	verifier.now = func() time.Time { return now }

	return verifier
}

func TestIssuedTokenIsVerified(t *testing.T) {
	token, err := newTestIssuer().Issue(testUserID, testEmail)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if !token.ExpiresAt.Equal(testNow.Add(time.Hour)) {
		t.Fatalf("Issue() expires at = %v", token.ExpiresAt)
	}

	userID, err := newTestVerifier(testNow).Verify(token.Value)
	if err != nil || userID != testUserID {
		t.Fatalf("Verify() = %q, %v", userID, err)
	}
}

func TestVerifierRejectsExpiredToken(t *testing.T) {
	token, err := newTestIssuer().Issue(testUserID, testEmail)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, err = newTestVerifier(testNow.Add(time.Hour + time.Second)).Verify(token.Value)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Verify() error = %v, want ErrExpiredToken", err)
	}
}

func TestVerifierRejectsInvalidTokens(t *testing.T) {
	foreignAlgorithm, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims{
		Email: testEmail,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   testUserID,
			Issuer:    TokenIssuer,
			ExpiresAt: jwt.NewNumericDate(testNow.Add(time.Hour)),
		},
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	for _, token := range []string{"not-a-jwt", foreignAlgorithm} {
		if _, err := newTestVerifier(testNow).Verify(token); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("Verify(%q) error = %v, want ErrInvalidToken", token, err)
		}
	}
}
