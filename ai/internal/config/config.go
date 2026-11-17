package config

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultGatewayURL         = "http://localhost:8080"
	defaultGatewayTimeout     = 10 * time.Second
	defaultLLMModel           = "gemini-3.5-flash-lite"
	defaultLLMMaxOutputTokens = 1024
	defaultLLMThinkingLevel   = "high"
	defaultLLMMinInterval     = 6 * time.Second
	defaultLLMRequestTimeout  = time.Minute
	defaultAgentTokenBudget   = 30000
	defaultAgentMaxSteps      = 8
	defaultUserInputMaxChars  = 2000
	defaultToolResultMaxBytes = 4000
	defaultMCPHTTPAddress     = ":8090"
	defaultMCPRateLimit       = 30
	defaultShutdownTimeout    = 10 * time.Second
	defaultProducerAgentURL   = "http://localhost:8091"
	defaultA2ACallTimeout     = 90 * time.Second
	defaultGrantEndpoint      = "http://localhost:8180/realms/land-plot/protocol/openid-connect/token"
	defaultAIClientID         = "land-plot-ai"
	defaultAIScopes           = "plots:read plots:write reports:read reports:write"
	minimumMCPAccessToken     = 32
)

var reasoningThinkingLevels = []string{"low", "medium", "high"}

type Gateway struct {
	URL          string
	Timeout      time.Duration
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

type LLM struct {
	Model                  string
	APIKey                 string
	MaxOutputTokens        int
	ReasoningThinkingLevel string
	MinInterval            time.Duration
	RequestTimeout         time.Duration
}

type Limits struct {
	TokenBudget        int
	MaxSteps           int
	UserInputMaxChars  int
	ToolResultMaxBytes int
}

type MCP struct {
	HTTPAddress        string
	AccessToken        string
	RateLimitPerMinute int
}

type AgentServer struct {
	HTTPAddress     string
	PublicURL       string
	ShutdownTimeout time.Duration
}

type Producer struct {
	URL         string
	CallTimeout time.Duration
}

func LoadGateway() (Gateway, error) {
	gateway := Gateway{}

	var err error
	if gateway.URL, err = httpURLOrDefault(envGatewayURL, defaultGatewayURL); err != nil {
		return Gateway{}, err
	}

	if gateway.Timeout, err = durationOrDefault(envGatewayTimeout, defaultGatewayTimeout); err != nil {
		return Gateway{}, err
	}

	if gateway.TokenURL, err = httpURLOrDefault(envOIDCTokenURL, defaultGrantEndpoint); err != nil {
		return Gateway{}, err
	}

	if gateway.ClientSecret, err = requiredString(envAIClientSecret); err != nil {
		return Gateway{}, err
	}

	gateway.ClientID = stringOrDefault(envAIClientID, defaultAIClientID)
	if gateway.ClientID == "" {
		return Gateway{}, fmt.Errorf(formatMustNotBeEmpty, envAIClientID)
	}

	gateway.Scopes = strings.Fields(stringOrDefault(envAIScopes, defaultAIScopes))
	if len(gateway.Scopes) == 0 {
		return Gateway{}, fmt.Errorf(formatMustNotBeEmpty, envAIScopes)
	}

	return gateway, nil
}

func LoadLLM() (LLM, error) {
	llm := LLM{
		Model:  stringOrDefault(envLLMModel, defaultLLMModel),
		APIKey: stringOrDefault(envLLMAPIKey, ""),
	}
	if llm.Model == "" {
		return LLM{}, fmt.Errorf(formatMustNotBeEmpty, envLLMModel)
	}

	level, err := oneOfOrDefault(envLLMThinkingLevel, defaultLLMThinkingLevel, reasoningThinkingLevels)
	if err != nil {
		return LLM{}, err
	}

	llm.ReasoningThinkingLevel = level

	if err := loadInts([]intSetting{
		{envLLMMaxOutputTokens, &llm.MaxOutputTokens, defaultLLMMaxOutputTokens},
	}); err != nil {
		return LLM{}, err
	}

	if err := loadDurations([]durationSetting{
		{envLLMMinInterval, &llm.MinInterval, defaultLLMMinInterval},
		{envLLMRequestTimeout, &llm.RequestTimeout, defaultLLMRequestTimeout},
	}); err != nil {
		return LLM{}, err
	}

	return llm, nil
}

func LoadLimits() (Limits, error) {
	limits := Limits{}
	if err := loadInts([]intSetting{
		{envAgentTokenBudget, &limits.TokenBudget, defaultAgentTokenBudget},
		{envAgentMaxSteps, &limits.MaxSteps, defaultAgentMaxSteps},
		{envUserInputMaxChars, &limits.UserInputMaxChars, defaultUserInputMaxChars},
		{envToolResultMaxBytes, &limits.ToolResultMaxBytes, defaultToolResultMaxBytes},
	}); err != nil {
		return Limits{}, err
	}

	return limits, nil
}

func LoadMCP() (MCP, error) {
	mcp := MCP{}

	var err error
	if mcp.RateLimitPerMinute, err = positiveIntOrDefault(envMCPRateLimitPerMinute, defaultMCPRateLimit); err != nil {
		return MCP{}, err
	}

	return mcp, nil
}

func LoadMCPHTTP() (MCP, error) {
	mcp, err := LoadMCP()
	if err != nil {
		return MCP{}, err
	}

	if mcp.HTTPAddress, err = listenAddressOrDefault(envMCPHTTPAddress, defaultMCPHTTPAddress); err != nil {
		return MCP{}, err
	}

	if mcp.AccessToken, err = requiredString(envMCPAccessToken); err != nil {
		return MCP{}, err
	}

	if len(mcp.AccessToken) < minimumMCPAccessToken {
		return MCP{}, fmt.Errorf(formatMustContainAtLeastCharacters, envMCPAccessToken, minimumMCPAccessToken)
	}

	return mcp, nil
}

func LoadAgentServer(defaultAddress, defaultPublicURL string) (AgentServer, error) {
	server := AgentServer{}

	var err error
	if server.HTTPAddress, err = listenAddressOrDefault(envAgentHTTPAddress, defaultAddress); err != nil {
		return AgentServer{}, err
	}

	if server.PublicURL, err = httpURLOrDefault(envAgentPublicURL, defaultPublicURL); err != nil {
		return AgentServer{}, err
	}

	if server.ShutdownTimeout, err = durationOrDefault(envHTTPShutdownTimeout, defaultShutdownTimeout); err != nil {
		return AgentServer{}, err
	}

	return server, nil
}

func LoadProducer() (Producer, error) {
	producer := Producer{}

	var err error
	if producer.URL, err = httpURLOrDefault(envProducerAgentURL, defaultProducerAgentURL); err != nil {
		return Producer{}, err
	}

	if producer.CallTimeout, err = durationOrDefault(envA2ACallTimeout, defaultA2ACallTimeout); err != nil {
		return Producer{}, err
	}

	return producer, nil
}
