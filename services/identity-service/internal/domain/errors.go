package domain

import "fmt"

var (
	ErrInvalidEmail = NewError(KindValidation, "invalid_email",
		"email address is invalid")

	ErrWeakPassword = NewError(KindValidation, "weak_password",
		fmt.Sprintf("password must contain at least %d characters", MinimumPasswordLength))

	ErrUserExists = NewError(KindConflict, "user_exists",
		"user with this email already exists")

	ErrUserNotFound = NewError(KindNotFound, "user_not_found",
		"user not found")

	ErrInvalidCredentials = NewError(KindUnauthorized, "invalid_credentials",
		"email or password is incorrect")
)
