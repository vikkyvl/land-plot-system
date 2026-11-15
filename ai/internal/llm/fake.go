package llm

import (
	"context"
	"slices"
	"sync"
)

type FakeTurn struct {
	Response Response
	Err      error
}

type Fake struct {
	mutex    sync.Mutex
	turns    []FakeTurn
	requests []Request
}

func NewFake(turns ...FakeTurn) *Fake {
	return &Fake{turns: turns}
}

func (fake *Fake) Generate(_ context.Context, request Request) (Response, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	request.History = slices.Clone(request.History)
	fake.requests = append(fake.requests, request)

	if len(fake.turns) == 0 {
		return Response{}, ErrNoScriptedResponse
	}

	turn := fake.turns[0]
	fake.turns = fake.turns[1:]

	return turn.Response, turn.Err
}

func (fake *Fake) Requests() []Request {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return slices.Clone(fake.requests)
}

func ReplyText(text string, usage Usage) FakeTurn {
	return FakeTurn{Response: Response{Message: Message{Role: RoleModel, Text: text}, Usage: usage}}
}

func ReplyToolCalls(usage Usage, calls ...ToolCall) FakeTurn {
	return FakeTurn{Response: Response{Message: Message{Role: RoleModel, ToolCalls: calls}, Usage: usage}}
}

type Unavailable struct {
	Err error
}

func (provider Unavailable) Generate(context.Context, Request) (Response, error) {
	return Response{}, provider.Err
}
