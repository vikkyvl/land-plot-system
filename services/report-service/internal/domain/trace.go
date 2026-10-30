package domain

type TraceHop struct {
	Service  string
	Instance string
	Protocol string
}

type Trace struct {
	Message string
	Path    []TraceHop
}
