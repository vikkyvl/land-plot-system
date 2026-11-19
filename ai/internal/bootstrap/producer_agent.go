package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/ai/internal/a2aagent"
	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	producerAddress   = ":8091"
	producerPublicURL = "http://localhost:8091"
	producerService   = "producer-agent"
)

type agentSettings struct {
	gateway config.Gateway
	llm     config.LLM
	limits  config.Limits
	server  config.AgentServer
}

func RunProducerAgent(runContext context.Context, stderr io.Writer) int {
	logger := newLogger(stderr, producerService, slog.LevelInfo)

	settings, err := loadAgentSettings(producerAddress, producerPublicURL)
	if err != nil {
		logger.Error(logLoadConfig, logging.KeyError, err)
		return ExitError
	}

	landClient := newLandClient(settings.gateway)
	if err := landClient.Authenticate(runContext); err != nil {
		logger.Warn(logAuthenticateLater, logging.KeyError, err)
	}

	toolset := tools.NewToolset(
		landClient,
		tools.Options{MaxResultBytes: settings.limits.ToolResultMaxBytes},
		tools.ProducerNames...)
	assistant := newTaskAgent(runContext, settings, agent.ProducerPrompt(), agent.ToolsFrom(toolset), logger)
	executor := a2aagent.NewExecutor(assistant, a2aagent.DecideProducer, logger)

	return serveAgent(
		runContext,
		settings.server,
		logging.AccessLog(
			logger,
			logging.PatternRoute,
			a2aagent.NewHandler(a2aagent.ProducerCard(settings.server.PublicURL), executor),
		),
		logger,
	)
}

func loadAgentSettings(defaultAddress, defaultPublicURL string) (agentSettings, error) {
	gateway, err := config.LoadGateway()
	if err != nil {
		return agentSettings{}, err
	}

	llmSettings, err := config.LoadLLM()
	if err != nil {
		return agentSettings{}, err
	}

	limits, err := config.LoadLimits()
	if err != nil {
		return agentSettings{}, err
	}

	server, err := config.LoadAgentServer(defaultAddress, defaultPublicURL)
	if err != nil {
		return agentSettings{}, err
	}

	return agentSettings{gateway: gateway, llm: llmSettings, limits: limits, server: server}, nil
}

func newTaskAgent(
	runContext context.Context,
	settings agentSettings,
	systemPrompt string,
	agentTools []agent.Tool,
	logger *slog.Logger,
) *agent.Agent {
	return agent.New(
		newProvider(runContext, settings.llm, logger),
		agentTools,
		agent.Config{
			SystemPrompt:      systemPrompt,
			ThinkingLevel:     llm.ThinkingMinimal,
			TokenBudget:       settings.limits.TokenBudget,
			MaxSteps:          settings.limits.MaxSteps,
			UserInputMaxChars: settings.limits.UserInputMaxChars,
			Secrets:           []string{settings.gateway.ClientSecret, settings.llm.APIKey},
		},
		logger,
	)
}

func serveAgent(runContext context.Context, server config.AgentServer, handler http.Handler, logger *slog.Logger) int {
	httpServer := &http.Server{Addr: server.HTTPAddress, Handler: handler, ReadHeaderTimeout: readHeaderTimeout}

	logger.Info(logAgentStarted, logging.KeyAddress, server.HTTPAddress, logging.KeyURL, server.PublicURL)

	return serveUntilDone(runContext, httpServer, server.ShutdownTimeout, logger)
}
