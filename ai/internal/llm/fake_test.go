package llm

import (
	"context"
	"errors"
	"testing"
)

func TestFakeReturnsScriptedTurnsAndRecordsRequests(t *testing.T) {
	fake := NewFake(
		ReplyToolCalls(Usage{TotalTokens: 10}, ToolCall{ID: "1", Name: "list_plots"}),
		ReplyText("Готово", Usage{TotalTokens: 5}),
	)

	first, err := fake.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if err != nil || first.Message.ToolCalls[0].Name != "list_plots" || first.Usage.TotalTokens != 10 {
		t.Fatalf("first Generate() = %#v, %v", first, err)
	}

	second, err := fake.Generate(context.Background(), Request{System: "system"})
	if err != nil || second.Message.Text != "Готово" {
		t.Fatalf("second Generate() = %#v, %v", second, err)
	}

	if _, err := fake.Generate(context.Background(), Request{}); !errors.Is(err, ErrNoScriptedResponse) {
		t.Fatalf("third Generate() error = %v", err)
	}

	requests := fake.Requests()
	if len(requests) != 3 || requests[0].History[0].Text != "Привіт" || requests[1].System != "system" {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestUnavailableProviderReturnsItsError(t *testing.T) {
	_, err := Unavailable{Err: ErrMissingAPIKey}.Generate(context.Background(), Request{})
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("Generate() error = %v", err)
	}
}
