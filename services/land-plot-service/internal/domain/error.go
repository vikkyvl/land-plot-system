package domain

import "errors"

type ErrorKind int

const (
	KindInvalidRequest ErrorKind = iota + 1
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindGone
	KindUnavailable
)

type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
	Details any
}

func NewError(kind ErrorKind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func (domainError *Error) Error() string {
	return domainError.Message
}

func (domainError *Error) WithDetails(details any) *Error {
	detailed := *domainError
	detailed.Details = details

	return &detailed
}

func (domainError *Error) Is(target error) bool {
	targetError, ok := target.(*Error)

	return ok && targetError.Kind == domainError.Kind && targetError.Code == domainError.Code
}

func ErrorCodes(errs ...error) []string {
	codes := make([]string, 0, len(errs))

	for _, err := range errs {
		var domainError *Error
		if errors.As(err, &domainError) {
			codes = append(codes, domainError.Code)
		}
	}

	return codes
}
