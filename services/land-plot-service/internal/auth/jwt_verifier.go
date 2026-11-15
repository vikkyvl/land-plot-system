package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenIssuer = "land-plot-identity"

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type JWTVerifier struct {
	secret []byte
	now    func() time.Time
}

func NewJWTVerifier(secret string) *JWTVerifier {
	return &JWTVerifier{secret: []byte(secret), now: time.Now}
}

func (verifier *JWTVerifier) Verify(token string) (string, error) {
	var tokenClaims claims

	parsed, err := jwt.ParseWithClaims(token, &tokenClaims,
		func(*jwt.Token) (any, error) { return verifier.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(verifier.now),
	)
	if errors.Is(err, jwt.ErrTokenExpired) {
		return "", fmt.Errorf("%w: %w", ErrExpiredToken, err)
	}

	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if !parsed.Valid || tokenClaims.Subject == "" || tokenClaims.Email == "" {
		return "", ErrInvalidToken
	}

	return tokenClaims.Subject, nil
}
