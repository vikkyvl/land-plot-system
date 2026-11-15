package auth

import "github.com/golang-jwt/jwt/v5"

const TokenIssuer = "land-plot-identity"

var signingMethod = jwt.SigningMethodHS256

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}
