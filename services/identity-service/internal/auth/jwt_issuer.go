package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/service"
)

type JWTIssuer struct {
	secret        []byte
	tokenLifetime time.Duration
	now           func() time.Time
}

func NewJWTIssuer(secret string, tokenLifetime time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), tokenLifetime: tokenLifetime, now: time.Now}
}

func (issuer *JWTIssuer) Issue(userID, email string) (service.Token, error) {
	issuedAt := issuer.now().UTC()
	expiresAt := issuedAt.Add(issuer.tokenLifetime)
	tokenClaims := claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    TokenIssuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	value, err := jwt.NewWithClaims(signingMethod, tokenClaims).SignedString(issuer.secret)
	if err != nil {
		return service.Token{}, fmt.Errorf(wrapSignJWT, err)
	}

	return service.Token{Value: value, ExpiresAt: expiresAt}, nil
}
