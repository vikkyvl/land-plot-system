package bootstrap

import (
	"context"
	"io"
	"log/slog"

	"github.com/vleshchenko/land-plot-system/ai/internal/a2aagent"
	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	consumerAddress   = ":8092"
	consumerPublicURL = "http://localhost:8092"
	consumerService   = "consumer-agent"
)

func RunConsumerAgent(runContext context.Context, stderr io.Writer) int {
	logger := newLogger(stderr, consumerService, slog.LevelInfo)
	defer startTracing(consumerService, logger)()

	settings, err := loadAgentSettings(consumerAddress, consumerPublicURL)
	if err != nil {
		logger.Error(logLoadConfig, logging.KeyError, err)
		return ExitError
	}

	producer, err := config.LoadProducer()
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
		tools.ConsumerNames...,
	)
	delegate := a2aagent.NewProducerDelegate(producer.URL, producer.CallTimeout, logger)
	agentTools := append(agent.ToolsFrom(toolset), delegate.Tool())

	assistant := newTaskAgent(runContext, settings, agent.ConsumerPrompt(), agentTools, logger)
	executor := a2aagent.NewExecutor(assistant, a2aagent.DecideConsumer, logger)

	logger.Info(logTrustedProducer, logging.KeyURL, producer.URL)

	return serveAgent(
		runContext,
		settings.server,
		tracedHandler(logger, a2aagent.NewHandler(a2aagent.ConsumerCard(settings.server.PublicURL), executor)),
		logger,
	)
}
