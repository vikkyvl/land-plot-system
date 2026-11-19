package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
	"github.com/vleshchenko/land-plot-system/ai/internal/mcpserver"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	transportStdio    = "stdio"
	transportHTTP     = "http"
	readHeaderTimeout = 5 * time.Second
	mcpService        = "mcp-server"
)

func RunMCPServer(runContext context.Context, arguments []string, stderr io.Writer) int {
	transport, err := parseMCPTransport(arguments, stderr)
	if err != nil {
		_, _ = fmt.Fprint(stderr, formatMCPUsage)
		return ExitUsage
	}

	logger := newLogger(stderr, mcpService, slog.LevelInfo)

	gateway, err := config.LoadGateway()
	if err != nil {
		logger.Error(logLoadConfig, logging.KeyError, err)
		return ExitError
	}

	limits, err := config.LoadLimits()
	if err != nil {
		logger.Error(logLoadConfig, logging.KeyError, err)
		return ExitError
	}

	landClient := newLandClient(gateway)
	if err := landClient.Authenticate(runContext); err != nil {
		logger.Warn(logAuthenticateLater, logging.KeyError, err)
	}

	toolset := tools.NewToolset(landClient, tools.Options{MaxResultBytes: limits.ToolResultMaxBytes}, tools.AllNames...)

	if transport == transportStdio {
		return runMCPStdio(runContext, toolset, logger)
	}

	return runMCPHTTP(runContext, toolset, logger)
}

func parseMCPTransport(arguments []string, stderr io.Writer) (string, error) {
	flags := flag.NewFlagSet("mcp-server", flag.ContinueOnError)
	flags.SetOutput(stderr)

	transport := flags.String("transport", transportStdio, "stdio or http")
	if err := flags.Parse(arguments); err != nil {
		return "", err
	}

	if (*transport != transportStdio && *transport != transportHTTP) || flags.NArg() > 0 {
		return "", errUnknownTransport
	}

	return *transport, nil
}

func runMCPStdio(runContext context.Context, toolset *tools.Toolset, logger *slog.Logger) int {
	settings, err := config.LoadMCP()
	if err != nil {
		logger.Error(logLoadConfig, logging.KeyError, err)
		return ExitError
	}

	server := mcpserver.New(toolset, mcpserver.Options{RateLimitPerMinute: settings.RateLimitPerMinute}, logger)
	logger.Info(logMCPStarted, logging.KeyTransport, transportStdio)

	if err := server.Run(runContext, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error(logMCPStopped, logging.KeyError, err)
		return ExitError
	}

	return ExitOK
}

func runMCPHTTP(runContext context.Context, toolset *tools.Toolset, logger *slog.Logger) int {
	settings, err := config.LoadMCPHTTP()
	if err != nil {
		logger.Error(logLoadConfig, logging.KeyError, err)
		return ExitError
	}

	server := mcpserver.New(toolset, mcpserver.Options{RateLimitPerMinute: settings.RateLimitPerMinute}, logger)
	httpServer := &http.Server{
		Addr: settings.HTTPAddress,
		Handler: logging.AccessLog(
			logger,
			logging.PatternRoute,
			mcpserver.NewHTTPHandler(server, settings.AccessToken),
		),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	logger.Info(logMCPStarted,
		logging.KeyTransport, transportHTTP,
		logging.KeyAddress, settings.HTTPAddress,
		logging.KeyPath, mcpserver.MCPPath,
	)

	return serveUntilDone(runContext, httpServer, defaultShutdownTimeout, logger)
}

func serveUntilDone(
	runContext context.Context,
	httpServer *http.Server,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) int {
	serveErrors := make(chan error, 1)

	go func() {
		serveErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErrors:
		logger.Error(logHTTPServerStopped, logging.KeyError, err)
		return ExitError
	case <-runContext.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(runContext), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownContext); err != nil {
		logger.Error(logHTTPServerStopped, logging.KeyError, err)
		return ExitError
	}

	return ExitOK
}
