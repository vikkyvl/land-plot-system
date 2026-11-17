package mcpserver

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	serverName    = "land-plot-mcp"
	serverTitle   = "Land plot system"
	serverVersion = "1.0.0"
	mimeTypeJSON  = "application/json"
)

type Options struct {
	RateLimitPerMinute int
}

type service struct {
	toolset *tools.Toolset
	limiter *rateLimiter
	logger  *slog.Logger
}

func New(toolset *tools.Toolset, options Options, logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: serverName, Title: serverTitle, Version: serverVersion},
		&mcp.ServerOptions{Instructions: serverInstructions},
	)

	handlers := &service{toolset: toolset, limiter: newRateLimiter(options.RateLimitPerMinute), logger: logger}
	handlers.registerTools(server)
	handlers.registerResources(server)
	handlers.registerPrompts(server)

	return server
}
