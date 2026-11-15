package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

const defaultShutdownTimeout = 10 * time.Second

func newLogger(output io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level}))
}

func newLandClient(gateway config.Gateway) *landapi.Client {
	return landapi.New(landapi.Config{
		BaseURL: gateway.URL,
		Timeout: gateway.Timeout,
		Credentials: landapi.Credentials{
			Email:    gateway.Email,
			Password: gateway.Password,
		},
	})
}

func newProvider(providerContext context.Context, settings config.LLM, logger *slog.Logger) llm.Provider {
	gemini, err := llm.NewGemini(providerContext, llm.GeminiConfig{
		APIKey:          settings.APIKey,
		Model:           settings.Model,
		MaxOutputTokens: settings.MaxOutputTokens,
		MinInterval:     settings.MinInterval,
		RequestTimeout:  settings.RequestTimeout,
	}, logger)
	if err != nil {
		logger.Warn(logLLMUnavailable, logging.KeyError, err)
		return llm.Unavailable{Err: err}
	}

	return gemini
}
