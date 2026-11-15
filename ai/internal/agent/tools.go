package agent

import (
	"context"
	"encoding/json"

	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

type ToolOutput struct {
	Text    string
	IsError bool
}

type Tool struct {
	Declaration llm.ToolDeclaration
	Run         func(callContext context.Context, arguments json.RawMessage) ToolOutput
}

func ToolsFrom(toolset *tools.Toolset) []Tool {
	definitions := toolset.Definitions()
	agentTools := make([]Tool, 0, len(definitions))

	for _, definition := range definitions {
		name := definition.Name
		agentTools = append(agentTools, Tool{
			Declaration: llm.ToolDeclaration{
				Name:        name,
				Description: definition.Description,
				Parameters:  definition.InputSchema,
			},
			Run: func(callContext context.Context, arguments json.RawMessage) ToolOutput {
				result := toolset.Call(callContext, name, arguments)
				return ToolOutput{Text: result.Text, IsError: result.IsError}
			},
		})
	}

	return agentTools
}
