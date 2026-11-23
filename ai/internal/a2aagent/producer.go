package a2aagent

import (
	"encoding/json"
	"fmt"

	"github.com/a2aproject/a2a-go/v2/a2a"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	SkillCreatePlot     = "create_plot"
	SkillUpdatePlot     = "update_plot"
	producerArtifact    = "plot"
	outcomeExistingPlot = "existing"
)

type plotView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version int64  `json:"version"`
}

type waitView struct {
	Outcome string    `json:"outcome"`
	Plot    *plotView `json:"plot"`
}

type listView struct {
	Count int        `json:"count"`
	Plots []plotView `json:"plots"`
}

func ProducerCard(publicURL string) *a2a.AgentCard {
	return newCard(producerName, producerDescription, publicURL, []a2a.AgentSkill{
		{
			ID:          SkillCreatePlot,
			Name:        "Create land plot",
			Description: skillCreatePlotDescription,
			Tags:        []string{"land-plot", "saga", "plot.created"},
			Examples:    []string{skillCreatePlotExample},
		},
		{
			ID:          SkillUpdatePlot,
			Name:        "Update land plot",
			Description: skillUpdatePlotDescription,
			Tags:        []string{"land-plot", "saga", "plot.updated"},
			Examples:    []string{skillUpdatePlotExample},
		},
	})
}

type producerTrace struct {
	sagaStarted bool
	waited      *waitView
	seen        *plotView
}

func DecideProducer(steps []agent.Step, result agent.Result, err error) Outcome {
	switch {
	case err != nil:
		return Outcome{Message: err.Error()}
	case result.StopReason != nil:
		return Outcome{Message: result.Answer}
	}

	trace := traceProducer(steps)

	switch {
	case trace.waited != nil && trace.waited.Outcome == tools.OutcomeConfirmed && trace.waited.Plot != nil:
		return completedPlot(*trace.waited.Plot, tools.OutcomeConfirmed, result.Answer)
	case trace.waited != nil:
		return Outcome{Message: failureMessage(fmt.Sprintf(formatSagaOutcome, trace.waited.Outcome), result.Answer)}
	case trace.sagaStarted:
		return Outcome{Message: failureMessage(reasonSagaNotAwaited, result.Answer)}
	case trace.seen != nil:
		return completedPlot(*trace.seen, outcomeExistingPlot, result.Answer)
	}

	return Outcome{Message: failureMessage(reasonNoPlot, result.Answer)}
}

func traceProducer(steps []agent.Step) producerTrace {
	var trace producerTrace

	for _, step := range steps {
		if step.Kind != agent.StepTool || step.IsError {
			continue
		}

		switch step.ToolName {
		case tools.CreatePlot, tools.UpdatePlot:
			trace.sagaStarted, trace.waited = true, nil
		case tools.WaitPlotConfirmed:
			var waited waitView
			if json.Unmarshal([]byte(step.Output), &waited) == nil && waited.Outcome != "" {
				trace.waited = &waited
			}
		case tools.GetPlot:
			trace.seen = decodePlot(step.Output, trace.seen)
		case tools.ListPlots:
			var list listView
			if json.Unmarshal([]byte(step.Output), &list) == nil && list.Count == 1 {
				trace.seen = &list.Plots[0]
			}
		}
	}

	return trace
}

func decodePlot(output string, fallback *plotView) *plotView {
	var plot plotView
	if json.Unmarshal([]byte(output), &plot) != nil || plot.ID == "" {
		return fallback
	}

	return &plot
}

func completedPlot(plot plotView, outcome, answer string) Outcome {
	return Outcome{
		Completed:    true,
		Message:      answer,
		ArtifactName: producerArtifact,
		Artifact: map[string]any{
			"plot_id": plot.ID,
			"name":    plot.Name,
			"status":  plot.Status,
			"version": plot.Version,
			"outcome": outcome,
		},
	}
}

func failureMessage(reason, answer string) string {
	if answer == "" {
		return reason
	}

	return reason + ". " + answer
}
