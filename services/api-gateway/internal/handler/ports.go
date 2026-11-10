package handler

type TokenVerifier interface {
	Verify(token string) (userID, email string, err error)
}
