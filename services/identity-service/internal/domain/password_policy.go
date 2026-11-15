package domain

const MinimumPasswordLength = 8

func ValidatePassword(password string) error {
	if len(password) < MinimumPasswordLength {
		return ErrWeakPassword
	}

	return nil
}
