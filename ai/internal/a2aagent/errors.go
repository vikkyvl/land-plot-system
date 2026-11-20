package a2aagent

import "errors"

var (
	errUnexpectedProducerReply   = errors.New("producer agent replied with a message instead of a task")
	errTrailingDelegateArguments = errors.New("arguments must contain exactly one JSON object")
)
