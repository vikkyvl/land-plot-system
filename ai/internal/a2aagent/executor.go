package a2aagent

import (
	"context"
	"encoding/json"
	"iter"
	"log/slog"
	"strings"
	"sync"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

type Runner interface {
	Run(runContext context.Context, userInput string, observe agent.Observer) (agent.Result, error)
}

type Outcome struct {
	Completed    bool
	Message      string
	ArtifactName string
	Artifact     map[string]any
}

type Decide func(steps []agent.Step, result agent.Result, err error) Outcome

type Executor struct {
	runner Runner
	decide Decide
	logger *slog.Logger
}

func NewExecutor(runner Runner, decide Decide, logger *slog.Logger) *Executor {
	return &Executor{runner: runner, decide: decide, logger: logger}
}

func (executor *Executor) Execute(
	executeContext context.Context,
	execution *a2asrv.ExecutorContext,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execution.StoredTask == nil && !yield(a2a.NewSubmittedTask(execution, execution.Message), nil) {
			return
		}

		if !yield(a2a.NewStatusUpdateEvent(execution, a2a.TaskStateWorking, nil), nil) {
			return
		}

		var (
			stepsMutex sync.Mutex
			steps      []agent.Step
		)

		result, err := executor.runner.Run(executeContext, TaskText(execution.Message), func(step agent.Step) {
			stepsMutex.Lock()
			defer stepsMutex.Unlock()

			steps = append(steps, step)
		})

		outcome := executor.decide(steps, result, err)
		executor.logOutcome(execution, outcome, result, err)

		if outcome.Artifact != nil {
			artifact := a2a.NewArtifactEvent(execution, a2a.NewDataPart(outcome.Artifact))
			artifact.Artifact.Name = outcome.ArtifactName

			if !yield(artifact, nil) {
				return
			}
		}

		state := a2a.TaskStateFailed
		if outcome.Completed {
			state = a2a.TaskStateCompleted
		}

		reply := a2a.NewMessageForTask(a2a.MessageRoleAgent, execution, a2a.NewTextPart(outcome.Message))
		yield(a2a.NewStatusUpdateEvent(execution, state, reply), nil)
	}
}

func (executor *Executor) Cancel(_ context.Context, execution *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execution.StoredTask == nil {
			yield(nil, a2a.ErrTaskNotFound)
			return
		}

		yield(a2a.NewStatusUpdateEvent(execution, a2a.TaskStateCanceled, nil), nil)
	}
}

func (executor *Executor) logOutcome(
	execution *a2asrv.ExecutorContext,
	outcome Outcome,
	result agent.Result,
	err error,
) {
	attributes := []any{
		logging.KeyTaskID, string(execution.TaskID),
		logging.KeyCompleted, outcome.Completed,
		logging.KeySteps, result.Steps,
		logging.KeyInputTokens, result.Usage.InputTokens,
		logging.KeyOutputTokens, result.Usage.OutputTokens,
		logging.KeyTotalTokens, result.Usage.TotalTokens,
	}
	if err != nil {
		attributes = append(attributes, logging.KeyError, err)
	}

	executor.logger.Info(logTaskFinished, attributes...)
}

func TaskText(message *a2a.Message) string {
	if message == nil {
		return ""
	}

	parts := make([]string, 0, len(message.Parts))

	for _, part := range message.Parts {
		if text := part.Text(); text != "" {
			parts = append(parts, text)
			continue
		}

		if data := part.Data(); data != nil {
			if encoded, err := json.Marshal(data); err == nil {
				parts = append(parts, taskDataPrefix+string(encoded))
			}
		}
	}

	return strings.Join(parts, "\n")
}
