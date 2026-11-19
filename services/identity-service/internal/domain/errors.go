package domain

var ErrInvalidSubject = NewError(KindUnauthorized, "invalid_subject",
	"access token subject must be a UUID")
