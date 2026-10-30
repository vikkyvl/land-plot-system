package auth

import "errors"

var (
	ErrInvalidToken = errors.New("invalid access token")
	ErrExpiredToken = errors.New("access token expired")
)
