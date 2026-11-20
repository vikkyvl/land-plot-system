package llm

import (
	"context"
	"encoding/json"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleModel Role = "model"
	RoleTool  Role = "tool"
)

type ThinkingLevel string

const (
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow     ThinkingLevel = "low"
	ThinkingMedium  ThinkingLevel = "medium"
	ThinkingHigh    ThinkingLevel = "high"
)

type ToolDeclaration struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type ToolResult struct {
	CallID  string
	Name    string
	Content string
	IsError bool
}

type Message struct {
	Role        Role
	Text        string
	ToolCalls   []ToolCall
	ToolResults []ToolResult

	native any
}

type Usage struct {
	InputTokens    int
	OutputTokens   int
	ThinkingTokens int
	TotalTokens    int
}

type Request struct {
	System        string
	History       []Message
	Tools         []ToolDeclaration
	ThinkingLevel ThinkingLevel
}

type Response struct {
	Message Message
	Usage   Usage
}

type Provider interface {
	Generate(requestContext context.Context, request Request) (Response, error)
}

func UserMessage(text string) Message {
	return Message{Role: RoleUser, Text: text}
}

func ToolResultsMessage(results ...ToolResult) Message {
	return Message{Role: RoleTool, ToolResults: results}
}
