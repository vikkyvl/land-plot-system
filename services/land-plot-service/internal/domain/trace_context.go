package domain

type TraceContext map[string]string

type ExpiredSaga struct {
	ID           string
	TraceContext TraceContext
}
