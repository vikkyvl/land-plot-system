package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

const requestIDBytes = 16

type requestIDKey struct{}

func WithRequestID(parent context.Context, requestID string) context.Context {
	if requestID == "" {
		return parent
	}

	return context.WithValue(parent, requestIDKey{}, requestID)
}

func RequestID(source context.Context) string {
	requestID, _ := source.Value(requestIDKey{}).(string)

	return requestID
}

func NewRequestID() string {
	var value [requestIDBytes]byte

	_, _ = rand.Read(value[:])

	return hex.EncodeToString(value[:])
}
