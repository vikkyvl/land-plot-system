package llm

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"google.golang.org/genai"

	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

type GeminiConfig struct {
	APIKey          string
	Model           string
	MaxOutputTokens int
	MinInterval     time.Duration
	RequestTimeout  time.Duration
	BaseURL         string
}

type Gemini struct {
	models   *genai.Models
	config   GeminiConfig
	logger   *slog.Logger
	throttle *throttle
	retries  retryPolicy
	now      func() time.Time
}

var thinkingLevels = map[ThinkingLevel]genai.ThinkingLevel{
	ThinkingMinimal: genai.ThinkingLevelMinimal,
	ThinkingLow:     genai.ThinkingLevelLow,
	ThinkingMedium:  genai.ThinkingLevelMedium,
	ThinkingHigh:    genai.ThinkingLevelHigh,
}

func NewGemini(clientContext context.Context, config GeminiConfig, logger *slog.Logger) (*Gemini, error) {
	if config.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	client, err := genai.NewClient(clientContext, &genai.ClientConfig{
		APIKey:      config.APIKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPClient:  &http.Client{Timeout: config.RequestTimeout},
		HTTPOptions: genai.HTTPOptions{BaseURL: config.BaseURL},
	})
	if err != nil {
		return nil, fmt.Errorf(wrapCreateGeminiClient, err)
	}

	return &Gemini{
		models:   client.Models,
		config:   config,
		logger:   logger,
		throttle: newThrottle(config.MinInterval, time.Now, sleepContext),
		retries:  defaultRetryPolicy(sleepContext),
		now:      time.Now,
	}, nil
}

func (gemini *Gemini) Generate(requestContext context.Context, request Request) (Response, error) {
	contents, err := toContents(request.History)
	if err != nil {
		return Response{}, err
	}

	config := gemini.generateConfig(request)

	for attempt := 1; ; attempt++ {
		if err := gemini.throttle.wait(requestContext); err != nil {
			return Response{}, fmt.Errorf(wrapWaitBeforeCall, err)
		}

		started := gemini.now()

		response, err := gemini.models.GenerateContent(requestContext, gemini.config.Model, contents, config)
		if err == nil {
			return gemini.parse(response, gemini.now().Sub(started))
		}

		delay, retry, finalErr := gemini.retries.decide(err, attempt)
		if !retry {
			gemini.logFailure(finalErr, attempt)
			return Response{}, fmt.Errorf(wrapGenerateContent, gemini.config.Model, finalErr)
		}

		gemini.logger.Warn(logRetryAfterLimit, logging.KeyAttempt, attempt, logging.KeyDelay, delay.String())

		if err := gemini.retries.sleep(requestContext, delay); err != nil {
			return Response{}, fmt.Errorf(wrapWaitBeforeCall, err)
		}
	}
}

func (gemini *Gemini) generateConfig(request Request) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: outputTokenLimit(gemini.config.MaxOutputTokens),
	}

	if request.System != "" {
		config.SystemInstruction = genai.NewContentFromText(request.System, genai.RoleUser)
	}

	if level, known := thinkingLevels[request.ThinkingLevel]; known {
		config.ThinkingConfig = &genai.ThinkingConfig{ThinkingLevel: level}
	}

	if len(request.Tools) == 0 {
		return config
	}

	declarations := make([]*genai.FunctionDeclaration, 0, len(request.Tools))
	names := make([]string, 0, len(request.Tools))

	for _, tool := range request.Tools {
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:                 tool.Name,
			Description:          tool.Description,
			ParametersJsonSchema: tool.Parameters,
		})
		names = append(names, tool.Name)
	}

	config.Tools = []*genai.Tool{{FunctionDeclarations: declarations}}
	config.ToolConfig = &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{
		Mode:                 genai.FunctionCallingConfigModeValidated,
		AllowedFunctionNames: names,
	}}

	return config
}

func (gemini *Gemini) logFailure(err error, attempt int) {
	if isDailyQuota(err) {
		gemini.logger.Error(logDailyQuotaReached, logging.KeyModel, gemini.config.Model, logging.KeyError, err)
		return
	}

	gemini.logger.Error(logModelCallFailed,
		logging.KeyModel, gemini.config.Model,
		logging.KeyAttempt, attempt,
		logging.KeyError, err,
	)
}

func outputTokenLimit(tokens int) int32 {
	if tokens <= 0 || tokens > math.MaxInt32 {
		return math.MaxInt32
	}

	return int32(tokens)
}
