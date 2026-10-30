package domain

import (
	"net/mail"
	"strings"
)

const maximumEmailLength = 254

func NormalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > maximumEmailLength {
		return "", ErrInvalidEmail
	}

	return email, nil
}
