package a2aagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
	"github.com/vleshchenko/land-plot-system/ai/internal/tracing"
)

const (
	DelegateToProducer   = "delegate_to_producer"
	maximumRequestLength = 2000
	stateCompleted       = "completed"
	stateFailed          = "failed"
)

var delegatedSkills = []string{SkillCreatePlot, SkillUpdatePlot}

type delegateArguments struct {
	Skill   string `json:"skill"`
	Request string `json:"request"`
}

type delegateResult struct {
	ProducerState string         `json:"producer_state"`
	TaskID        string         `json:"task_id,omitempty"`
	Plot          map[string]any `json:"plot,omitempty"`
	Message       string         `json:"message,omitempty"`
}

type ProducerDelegate struct {
	producerURL string
	httpClient  *http.Client
	logger      *slog.Logger
}

func NewProducerDelegate(producerURL string, timeout time.Duration, logger *slog.Logger) *ProducerDelegate {
	return &ProducerDelegate{
		producerURL: producerURL,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: tracing.Transport(logging.ForwardRequestID(http.DefaultTransport)),
		},
		logger: logger,
	}
}

func (delegate *ProducerDelegate) Tool() agent.Tool {
	return agent.Tool{
		Declaration: llm.ToolDeclaration{
			Name:        DelegateToProducer,
			Description: descriptionDelegate,
			Parameters: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"skill", "request"},
				"properties": map[string]any{
					"skill": map[string]any{
						"type":        "string",
						"enum":        delegatedSkills,
						"description": descriptionDelegateSkill,
					},
					"request": map[string]any{"type": "string", "description": descriptionDelegateRequest},
				},
			},
		},
		Run: delegate.run,
	}
}

func (delegate *ProducerDelegate) run(callContext context.Context, raw json.RawMessage) agent.ToolOutput {
	arguments, err := decodeDelegateArguments(raw)
	if err != nil {
		return toolError(codeInvalidArguments, err.Error())
	}

	started := time.Now()

	result, err := delegate.send(callContext, arguments)
	if err != nil {
		delegate.logger.Warn(logDelegationFailed, logging.KeyError, err)
		return toolError(codeProducerUnavailable, err.Error())
	}

	delegate.logger.Info(logDelegated,
		logging.KeySkill, arguments.Skill,
		logging.KeyTaskID, result.TaskID,
		logging.KeyStatus, result.ProducerState,
		logging.KeyDurationMS, time.Since(started).Milliseconds(),
	)

	encoded, _ := json.Marshal(result)

	return agent.ToolOutput{Text: string(encoded), IsError: result.ProducerState != stateCompleted}
}

func (delegate *ProducerDelegate) send(
	callContext context.Context,
	arguments delegateArguments,
) (delegateResult, error) {
	card, err := agentcard.NewResolver(delegate.httpClient).Resolve(callContext, delegate.producerURL)
	if err != nil {
		return delegateResult{}, fmt.Errorf(wrapResolveCard, delegate.producerURL, err)
	}

	if !slices.ContainsFunc(card.Skills, func(skill a2a.AgentSkill) bool { return skill.ID == arguments.Skill }) {
		return delegateResult{}, fmt.Errorf(formatMissingSkill, arguments.Skill)
	}

	client, err := a2aclient.NewFromEndpoints(
		callContext,
		[]*a2a.AgentInterface{a2a.NewAgentInterface(delegate.producerURL, a2a.TransportProtocolJSONRPC)},
		a2aclient.WithJSONRPCTransport(delegate.httpClient),
	)
	if err != nil {
		return delegateResult{}, fmt.Errorf(wrapCreateClient, err)
	}
	defer func() { _ = client.Destroy() }()

	response, err := client.SendMessage(callContext, &a2a.SendMessageRequest{
		Message: a2a.NewMessage(a2a.MessageRoleUser,
			a2a.NewTextPart(arguments.Request),
			a2a.NewDataPart(map[string]any{"skill": arguments.Skill}),
		),
	})
	if err != nil {
		return delegateResult{}, fmt.Errorf(wrapSendTask, err)
	}

	task, ok := response.(*a2a.Task)
	if !ok {
		return delegateResult{}, errUnexpectedProducerReply
	}

	return summarizeTask(task), nil
}

func summarizeTask(task *a2a.Task) delegateResult {
	result := delegateResult{
		ProducerState: stateFailed,
		TaskID:        string(task.ID),
		Message:       TaskText(task.Status.Message),
	}
	if task.Status.State == a2a.TaskStateCompleted {
		result.ProducerState = stateCompleted
	}

	for _, artifact := range task.Artifacts {
		for _, part := range artifact.Parts {
			if data, ok := part.Data().(map[string]any); ok {
				result.Plot = data
			}
		}
	}

	return result
}

func decodeDelegateArguments(raw json.RawMessage) (delegateArguments, error) {
	var arguments delegateArguments

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&arguments); err != nil {
		return delegateArguments{}, fmt.Errorf(wrapDecodeDelegate, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return delegateArguments{}, errTrailingDelegateArguments
	}

	if !slices.Contains(delegatedSkills, arguments.Skill) {
		return delegateArguments{}, fmt.Errorf(formatUnknownSkill, arguments.Skill)
	}

	if err := guard.CheckUserInput(arguments.Request, maximumRequestLength); err != nil {
		return delegateArguments{}, err
	}

	return arguments, nil
}

func toolError(code, message string) agent.ToolOutput {
	encoded, _ := json.Marshal(map[string]any{"error": map[string]string{"code": code, "message": message}})
	return agent.ToolOutput{Text: string(encoded), IsError: true}
}
