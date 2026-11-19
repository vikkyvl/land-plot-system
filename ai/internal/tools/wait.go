package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

const statusConfirmed = "CONFIRMED"

func (toolset *Toolset) waitPlotConfirmed(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[waitPlotArguments](raw)
	if err == nil {
		err = arguments.validate()
	}

	if err != nil {
		return invalidArguments(err)
	}

	waitContext, cancel := context.WithTimeout(requestContext, time.Duration(*arguments.TimeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(toolset.pollInterval)
	defer ticker.Stop()

	for {
		plot, err := toolset.api.GetPlot(waitContext, arguments.PlotID)
		if result, done := toolset.sagaOutcome(plot, err, *arguments.ExpectedVersion); done {
			return result
		}

		select {
		case <-waitContext.Done():
			if requestContext.Err() != nil {
				return apiFailure(requestContext.Err())
			}

			return toolset.success(waitResult{Outcome: OutcomeStillPending})
		case <-ticker.C:
		}
	}
}

func (toolset *Toolset) sagaOutcome(plot landapi.Plot, err error, expectedVersion int64) (Result, bool) {
	switch {
	case landapi.IsNotFound(err):
		return toolset.success(waitResult{Outcome: OutcomeCancelled}), true
	case isClientError(err):
		return apiFailure(err), true
	case err != nil:
		return Result{}, false
	case plot.Status != statusConfirmed:
		return Result{}, false
	case plot.Version < expectedVersion:
		return encodeError(errorDetails{Code: codeVersionMismatch, Message: messageVersionMismatch}), true
	}

	summary := summarizePlot(plot)
	if plot.Version > expectedVersion {
		return toolset.success(waitResult{Outcome: OutcomeRolledBack, Plot: &summary}), true
	}

	return toolset.success(waitResult{Outcome: OutcomeConfirmed, Plot: &summary}), true
}

func isClientError(err error) bool {
	apiError, ok := errors.AsType[*landapi.APIError](err)
	return ok && apiError.Status < http.StatusInternalServerError
}
