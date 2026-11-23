package mcpserver

import (
	"context"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

const rateWindow = time.Minute

func (handlers *service) registerTools(server *mcp.Server) {
	notDestructive, closedWorld := false, false

	for _, definition := range handlers.toolset.Definitions() {
		server.AddTool(&mcp.Tool{
			Name:        definition.Name,
			Description: definition.Description,
			InputSchema: definition.InputSchema,
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    definition.ReadOnly,
				DestructiveHint: &notDestructive,
				OpenWorldHint:   &closedWorld,
			},
		}, handlers.callTool(definition.Name))
	}
}

func (handlers *service) callTool(name string) mcp.ToolHandler {
	return func(callContext context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		started := time.Now()

		if !handlers.limiter.allow() {
			handlers.logger.Warn(logToolRateLimited, logging.KeyTool, name)
			return textResult(rateLimitedResult, true), nil
		}

		result := handlers.toolset.Call(callContext, name, request.Params.Arguments)

		text, masked := guard.MaskInjections(result.Text)
		if masked {
			handlers.logger.Warn(logSuspiciousToolData, logging.KeyTool, name)
		}

		handlers.logger.Info(logToolCall,
			logging.KeyTool, name,
			logging.KeyIsError, result.IsError,
			logging.KeyDurationMS, time.Since(started).Milliseconds(),
		)

		return textResult(text, result.IsError), nil
	}
}

func textResult(text string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, IsError: isError}
}

type rateLimiter struct {
	mutex  sync.Mutex
	limit  int
	calls  []time.Time
	window time.Duration
	now    func() time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{limit: limit, window: rateWindow, now: time.Now}
}

func (limiter *rateLimiter) allow() bool {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	now := limiter.now()
	recent := limiter.calls[:0]

	for _, call := range limiter.calls {
		if now.Sub(call) < limiter.window {
			recent = append(recent, call)
		}
	}

	limiter.calls = recent
	if len(limiter.calls) >= limiter.limit {
		return false
	}

	limiter.calls = append(limiter.calls, now)

	return true
}
