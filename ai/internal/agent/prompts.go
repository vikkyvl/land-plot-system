package agent

import (
	_ "embed"
	"fmt"
)

type Mode string

const (
	ModeNonReasoning Mode = "non-reasoning"
	ModeReasoning    Mode = "reasoning"
)

var (
	//go:embed prompts/non_reasoning.txt
	nonReasoningPrompt string

	//go:embed prompts/reasoning.txt
	reasoningPrompt string

	//go:embed prompts/producer.txt
	producerPrompt string

	//go:embed prompts/consumer.txt
	consumerPrompt string
)

func ConsumerPrompt() string {
	return consumerPrompt
}

func ProducerPrompt() string {
	return producerPrompt
}

func ParseMode(value string) (Mode, error) {
	switch Mode(value) {
	case ModeNonReasoning, ModeReasoning:
		return Mode(value), nil
	default:
		return "", fmt.Errorf(formatUnknownMode, value, ModeNonReasoning, ModeReasoning)
	}
}

func SystemPrompt(mode Mode) string {
	if mode == ModeReasoning {
		return reasoningPrompt
	}

	return nonReasoningPrompt
}
