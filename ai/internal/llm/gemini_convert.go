package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

const (
	functionResponseOutput = "output"
	functionResponseError  = "error"
)

var blockedFinishReasons = map[genai.FinishReason]bool{
	genai.FinishReasonSafety:            true,
	genai.FinishReasonRecitation:        true,
	genai.FinishReasonBlocklist:         true,
	genai.FinishReasonProhibitedContent: true,
	genai.FinishReasonSPII:              true,
}

func toContents(history []Message) ([]*genai.Content, error) {
	contents := make([]*genai.Content, 0, len(history))

	for _, message := range history {
		content, err := toContent(message)
		if err != nil {
			return nil, err
		}

		contents = append(contents, content)
	}

	return contents, nil
}

func toContent(message Message) (*genai.Content, error) {
	if native, ok := message.native.(*genai.Content); ok && native != nil {
		return native, nil
	}

	switch message.Role {
	case RoleModel:
		return modelContent(message)
	case RoleTool:
		return toolResultsContent(message), nil
	default:
		return genai.NewContentFromText(message.Text, genai.RoleUser), nil
	}
}

func modelContent(message Message) (*genai.Content, error) {
	content := &genai.Content{Role: genai.RoleModel}
	if message.Text != "" {
		content.Parts = append(content.Parts, &genai.Part{Text: message.Text})
	}

	for _, call := range message.ToolCalls {
		arguments := map[string]any{}
		if len(call.Arguments) > 0 {
			if err := json.Unmarshal(call.Arguments, &arguments); err != nil {
				return nil, fmt.Errorf(wrapEncodeToolCall, call.Name, err)
			}
		}

		content.Parts = append(content.Parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{ID: call.ID, Name: call.Name, Args: arguments},
		})
	}

	return content, nil
}

func toolResultsContent(message Message) *genai.Content {
	content := &genai.Content{Role: genai.RoleUser}

	for _, result := range message.ToolResults {
		key := functionResponseOutput
		if result.IsError {
			key = functionResponseError
		}

		content.Parts = append(content.Parts, &genai.Part{FunctionResponse: &genai.FunctionResponse{
			ID:       result.CallID,
			Name:     result.Name,
			Response: map[string]any{key: result.Content},
		}})
	}

	return content
}

func (gemini *Gemini) parse(response *genai.GenerateContentResponse, duration time.Duration) (Response, error) {
	usage := usageOf(response.UsageMetadata)
	finishReason := genai.FinishReasonUnspecified

	if len(response.Candidates) > 0 {
		finishReason = response.Candidates[0].FinishReason
	}

	gemini.logger.Info(logModelCall,
		logging.KeyModel, gemini.config.Model,
		logging.KeyInputTokens, usage.InputTokens,
		logging.KeyOutputTokens, usage.OutputTokens,
		logging.KeyThinkingTokens, usage.ThinkingTokens,
		logging.KeyTotalTokens, usage.TotalTokens,
		logging.KeyDurationMS, duration.Milliseconds(),
		logging.KeyFinishReason, string(finishReason),
	)

	if len(response.Candidates) == 0 || response.Candidates[0].Content == nil {
		if response.PromptFeedback != nil && response.PromptFeedback.BlockReason != "" {
			return Response{Usage: usage}, fmt.Errorf(formatBlockedResponse, ErrResponseBlocked,
				response.PromptFeedback.BlockReason)
		}

		return Response{Usage: usage}, ErrEmptyResponse
	}

	message, err := toMessage(response.Candidates[0].Content)
	if err != nil {
		return Response{Usage: usage}, err
	}

	if message.Text == "" && len(message.ToolCalls) == 0 {
		return Response{Usage: usage}, emptyAnswerError(finishReason)
	}

	return Response{Message: message, Usage: usage}, nil
}

func toMessage(content *genai.Content) (Message, error) {
	message := Message{Role: RoleModel, native: content}

	var text strings.Builder

	for _, part := range content.Parts {
		if part.Text != "" && !part.Thought {
			text.WriteString(part.Text)
		}

		if part.FunctionCall == nil {
			continue
		}

		arguments, err := json.Marshal(part.FunctionCall.Args)
		if err != nil {
			return Message{}, fmt.Errorf(wrapEncodeToolCall, part.FunctionCall.Name, err)
		}

		message.ToolCalls = append(message.ToolCalls, ToolCall{
			ID:        part.FunctionCall.ID,
			Name:      part.FunctionCall.Name,
			Arguments: arguments,
		})
	}

	message.Text = strings.TrimSpace(text.String())

	return message, nil
}

func emptyAnswerError(finishReason genai.FinishReason) error {
	switch {
	case finishReason == genai.FinishReasonMaxTokens:
		return ErrOutputTruncated
	case blockedFinishReasons[finishReason]:
		return fmt.Errorf(formatBlockedResponse, ErrResponseBlocked, finishReason)
	default:
		return ErrEmptyResponse
	}
}

func usageOf(metadata *genai.GenerateContentResponseUsageMetadata) Usage {
	if metadata == nil {
		return Usage{}
	}

	usage := Usage{
		InputTokens:    int(metadata.PromptTokenCount + metadata.ToolUsePromptTokenCount),
		OutputTokens:   int(metadata.CandidatesTokenCount),
		ThinkingTokens: int(metadata.ThoughtsTokenCount),
		TotalTokens:    int(metadata.TotalTokenCount),
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens + usage.ThinkingTokens
	}

	return usage
}
