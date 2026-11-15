package a2aagent

import (
	"encoding/json"

	"github.com/a2aproject/a2a-go/v2/a2a"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	SkillPlotReport  = "plot_report"
	consumerArtifact = "report"
)

type reportView struct {
	ID           string  `json:"id"`
	PlotID       string  `json:"plot_id"`
	PlotName     string  `json:"plot_name"`
	PlotVersion  int64   `json:"plot_version"`
	AreaM2       float64 `json:"area_m2"`
	AreaHectares float64 `json:"area_hectares"`
}

func ConsumerCard(publicURL string) *a2a.AgentCard {
	return newCard(consumerName, consumerDescription, publicURL, []a2a.AgentSkill{{
		ID:          SkillPlotReport,
		Name:        "Plot area report",
		Description: skillPlotReportDescription,
		Tags:        []string{"report", "area", "a2a-delegation"},
		Examples:    []string{skillPlotReportExample},
	}})
}

func DecideConsumer(steps []agent.Step, result agent.Result, err error) Outcome {
	switch {
	case err != nil:
		return Outcome{Message: err.Error()}
	case result.StopReason != nil:
		return Outcome{Message: result.Answer}
	}

	var (
		report             *reportView
		producerFailure    string
		delegationAttempts int
	)

	for _, step := range steps {
		if step.Kind != agent.StepTool {
			continue
		}

		switch {
		case step.ToolName == DelegateToProducer:
			delegationAttempts++

			if step.IsError {
				producerFailure = step.Output
			}
		case step.ToolName == tools.CreateReport && !step.IsError:
			var created reportView
			if json.Unmarshal([]byte(step.Output), &created) == nil && created.ID != "" {
				report = &created
			}
		}
	}

	switch {
	case report != nil:
		return completedReport(*report, result.Answer)
	case producerFailure != "":
		return Outcome{Message: failureMessage(reasonProducerFailed, result.Answer)}
	case delegationAttempts > 0:
		return Outcome{Message: failureMessage(reasonNoReportAfterDelegation, result.Answer)}
	}

	return Outcome{Message: failureMessage(reasonNoReport, result.Answer)}
}

func completedReport(report reportView, answer string) Outcome {
	return Outcome{
		Completed:    true,
		Message:      answer,
		ArtifactName: consumerArtifact,
		Artifact: map[string]any{
			"report_id":     report.ID,
			"plot_id":       report.PlotID,
			"plot_name":     report.PlotName,
			"plot_version":  report.PlotVersion,
			"area_m2":       report.AreaM2,
			"area_hectares": report.AreaHectares,
		},
	}
}
