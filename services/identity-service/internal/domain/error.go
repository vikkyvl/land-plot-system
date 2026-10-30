package domain

type ErrorKind int

const (
	KindInvalidRequest ErrorKind = iota + 1
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindUnavailable
)

type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
}

func NewError(kind ErrorKind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func (domainError *Error) Error() string {
	return domainError.Message
}
