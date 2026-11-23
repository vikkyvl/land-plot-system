package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2

	fcClientService = "fc-client"
)

type fcOptions struct {
	mode    agent.Mode
	verbose bool
	query   string
}

type fcSettings struct {
	gateway config.Gateway
	llm     config.LLM
	limits  config.Limits
}

func RunFCClient(runContext context.Context, arguments []string, stdout, stderr io.Writer) int {
	options, err := parseFCOptions(arguments, stderr)
	if err != nil {
		_, _ = fmt.Fprint(stderr, formatUsage)
		return ExitUsage
	}

	settings, err := loadFCSettings()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return ExitError
	}

	level := slog.LevelWarn
	if options.verbose {
		level = slog.LevelInfo
	}

	logger := newLogger(stderr, fcClientService, level)
	defer startTracing(fcClientService, logger)()

	landClient := newLandClient(settings.gateway)
	if err := landClient.Authenticate(runContext); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return ExitError
	}

	toolset := tools.NewToolset(
		landClient,
		tools.Options{MaxResultBytes: settings.limits.ToolResultMaxBytes},
		tools.AllNames...)
	assistant := agent.New(
		newProvider(runContext, settings.llm, logger),
		agent.ToolsFrom(toolset),
		agent.Config{
			SystemPrompt:      agent.SystemPrompt(options.mode),
			ThinkingLevel:     thinkingLevel(options.mode, settings.llm),
			TokenBudget:       settings.limits.TokenBudget,
			MaxSteps:          settings.limits.MaxSteps,
			UserInputMaxChars: settings.limits.UserInputMaxChars,
			Secrets:           []string{settings.gateway.ClientSecret, settings.llm.APIKey},
		},
		logger,
	)

	printer := newConsole(stdout)
	printer.header(options.mode, settings.llm.Model, options.query)

	result, err := assistant.Run(runContext, options.query, printer.step)
	if err != nil {
		printer.failure(result, err)
		return ExitError
	}

	printer.result(result)

	return ExitOK
}

func parseFCOptions(arguments []string, stderr io.Writer) (fcOptions, error) {
	flags := flag.NewFlagSet("fc-client", flag.ContinueOnError)
	flags.SetOutput(stderr)

	mode := flags.String("mode", string(agent.ModeNonReasoning), "non-reasoning or reasoning")
	verbose := flags.Bool("v", false, "write JSON logs with token usage to stderr")

	if err := flags.Parse(arguments); err != nil {
		return fcOptions{}, err
	}

	parsedMode, err := agent.ParseMode(*mode)
	if err != nil {
		return fcOptions{}, err
	}

	query := strings.TrimSpace(strings.Join(flags.Args(), " "))
	if query == "" {
		return fcOptions{}, errMissingQuery
	}

	return fcOptions{mode: parsedMode, verbose: *verbose, query: query}, nil
}

func loadFCSettings() (fcSettings, error) {
	gateway, err := config.LoadGateway()
	if err != nil {
		return fcSettings{}, err
	}

	llmSettings, err := config.LoadLLM()
	if err != nil {
		return fcSettings{}, err
	}

	if llmSettings.APIKey == "" {
		return fcSettings{}, llm.ErrMissingAPIKey
	}

	limits, err := config.LoadLimits()
	if err != nil {
		return fcSettings{}, err
	}

	return fcSettings{gateway: gateway, llm: llmSettings, limits: limits}, nil
}

func thinkingLevel(mode agent.Mode, settings config.LLM) llm.ThinkingLevel {
	if mode == agent.ModeReasoning {
		return llm.ThinkingLevel(settings.ReasoningThinkingLevel)
	}

	return llm.ThinkingMinimal
}
