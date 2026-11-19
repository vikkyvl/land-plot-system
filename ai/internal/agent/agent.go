package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

type Config struct {
	SystemPrompt      string
	ThinkingLevel     llm.ThinkingLevel
	TokenBudget       int
	MaxSteps          int
	UserInputMaxChars int
	Secrets           []string
}

type StepKind string

const (
	StepModel StepKind = "model"
	StepTool  StepKind = "tool"
)

type Step struct {
	Kind      StepKind
	Usage     llm.Usage
	ToolName  string
	Arguments json.RawMessage
	Output    string
	IsError   bool
}

type Observer func(step Step)

type Result struct {
	Answer     string
	Steps      int
	Usage      llm.Usage
	StopReason error
}

type Agent struct {
	provider llm.Provider
	tools    map[string]Tool
	ordered  []llm.ToolDeclaration
	config   Config
	redactor *guard.SecretRedactor
	logger   *slog.Logger
}

func New(provider llm.Provider, agentTools []Tool, config Config, logger *slog.Logger) *Agent {
	byName := make(map[string]Tool, len(agentTools))
	declarations := make([]llm.ToolDeclaration, 0, len(agentTools))

	for _, tool := range agentTools {
		if _, duplicate := byName[tool.Declaration.Name]; duplicate {
			continue
		}

		byName[tool.Declaration.Name] = tool
		declarations = append(declarations, tool.Declaration)
	}

	return &Agent{
		provider: provider,
		tools:    byName,
		ordered:  declarations,
		config:   config,
		redactor: guard.NewSecretRedactor(config.Secrets...),
		logger:   logger,
	}
}

func (agent *Agent) Run(runContext context.Context, userInput string, observe Observer) (Result, error) {
	if err := guard.CheckUserInput(userInput, agent.config.UserInputMaxChars); err != nil {
		agent.logger.Warn(logUserInputRejected, logging.KeyError, err)
		return Result{}, err
	}

	if observe == nil {
		observe = func(Step) {}
	}

	run := &runState{
		history: []llm.Message{llm.UserMessage(userInput)},
		budget:  guard.NewTokenBudget(agent.config.TokenBudget),
		steps:   guard.NewStepLimiter(agent.config.MaxSteps),
	}

	for {
		if err := run.budget.Check(); err != nil {
			return agent.stop(run, guard.MessageBudgetExhausted, err), nil
		}

		response, err := agent.generate(runContext, run)
		observe(Step{Kind: StepModel, Usage: response.Usage})

		if err != nil {
			return run.result(""), err
		}

		run.history = append(run.history, response.Message)

		if len(response.Message.ToolCalls) == 0 {
			return run.result(agent.safeAnswer(response.Message.Text)), nil
		}

		if err := run.steps.Take(); err != nil {
			return agent.stop(run, messageStepLimitReached, err), nil
		}

		run.history = append(run.history, agent.callTools(runContext, response.Message.ToolCalls, observe))
	}
}

func (agent *Agent) generate(runContext context.Context, run *runState) (llm.Response, error) {
	response, err := agent.provider.Generate(runContext, llm.Request{
		System:        agent.config.SystemPrompt,
		History:       run.history,
		Tools:         agent.ordered,
		ThinkingLevel: agent.config.ThinkingLevel,
	})

	run.budget.Spend(response.Usage.TotalTokens)
	run.addUsage(response.Usage)

	agent.logger.Info(logAgentModelStep,
		logging.KeyStep, run.steps.Taken()+1,
		logging.KeyInputTokens, response.Usage.InputTokens,
		logging.KeyOutputTokens, response.Usage.OutputTokens,
		logging.KeyThinkingTokens, response.Usage.ThinkingTokens,
		logging.KeyTotalTokens, response.Usage.TotalTokens,
		logging.KeySpentTokens, run.budget.Spent(),
	)

	if err != nil {
		return response, fmt.Errorf(wrapModelStep, err)
	}

	return response, nil
}

func (agent *Agent) callTools(runContext context.Context, calls []llm.ToolCall, observe Observer) llm.Message {
	results := make([]llm.ToolResult, 0, len(calls))

	for _, call := range calls {
		output := agent.callTool(runContext, call)
		observe(
			Step{
				Kind:      StepTool,
				ToolName:  call.Name,
				Arguments: call.Arguments,
				Output:    output.Text,
				IsError:   output.IsError,
			},
		)

		results = append(results, llm.ToolResult{
			CallID:  call.ID,
			Name:    call.Name,
			Content: guard.WrapToolResult(call.Name, output.Text),
			IsError: output.IsError,
		})
	}

	return llm.ToolResultsMessage(results...)
}

func (agent *Agent) callTool(runContext context.Context, call llm.ToolCall) ToolOutput {
	tool, allowed := agent.tools[call.Name]
	if !allowed {
		agent.logger.Warn(logToolNotAllowed, logging.KeyTool, call.Name)
		return ToolOutput{Text: fmt.Sprintf(formatToolNotAllowed, call.Name), IsError: true}
	}

	started := time.Now()
	output := tool.Run(runContext, call.Arguments)

	masked, changed := guard.MaskInjections(output.Text)
	if changed {
		agent.logger.Warn(logSuspiciousToolData, logging.KeyTool, call.Name)
	}

	agent.logger.Info(logToolCall,
		logging.KeyTool, call.Name,
		logging.KeyIsError, output.IsError,
		logging.KeyDurationMS, time.Since(started).Milliseconds(),
	)

	return ToolOutput{Text: masked, IsError: output.IsError}
}

func (agent *Agent) safeAnswer(answer string) string {
	redacted, changed := agent.redactor.Redact(answer)
	if changed {
		agent.logger.Warn(logSecretRedacted)
	}

	return redacted
}

func (agent *Agent) stop(run *runState, answer string, reason error) Result {
	agent.logger.Warn(logAgentStopped, logging.KeyError, reason, logging.KeySpentTokens, run.budget.Spent())

	result := run.result(answer)
	result.StopReason = reason

	return result
}

func IsLimitStop(err error) bool {
	return errors.Is(err, guard.ErrTokenBudgetExhausted) || errors.Is(err, guard.ErrStepLimitReached)
}

type runState struct {
	history []llm.Message
	budget  *guard.TokenBudget
	steps   *guard.StepLimiter
	usage   llm.Usage
}

func (run *runState) addUsage(usage llm.Usage) {
	run.usage.InputTokens += usage.InputTokens
	run.usage.OutputTokens += usage.OutputTokens
	run.usage.ThinkingTokens += usage.ThinkingTokens
	run.usage.TotalTokens += usage.TotalTokens
}

func (run *runState) result(answer string) Result {
	return Result{Answer: answer, Steps: run.steps.Taken(), Usage: run.usage}
}
