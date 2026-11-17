package config

import (
	"strings"
	"testing"
	"time"
)

func setGatewayEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AI_CLIENT_SECRET", "ai-client-secret")
}

func TestLoadGatewayAppliesDefaults(t *testing.T) {
	setGatewayEnv(t)

	gateway, err := LoadGateway()
	if err != nil {
		t.Fatalf("LoadGateway() error = %v", err)
	}

	if gateway.URL != defaultGatewayURL || gateway.Timeout != defaultGatewayTimeout ||
		gateway.TokenURL != defaultGrantEndpoint || gateway.ClientID != defaultAIClientID ||
		gateway.ClientSecret != "ai-client-secret" ||
		strings.Join(gateway.Scopes, " ") != "plots:read plots:write reports:read reports:write" {
		t.Fatalf("LoadGateway() = %#v", gateway)
	}
}

func TestLoadGatewayReadsTheScopesOfTheProgram(t *testing.T) {
	setGatewayEnv(t)
	t.Setenv("AI_SCOPES", " plots:read   plots:write ")
	t.Setenv("OIDC_TOKEN_URL", "http://keycloak:8080/realms/land-plot/protocol/openid-connect/token")

	gateway, err := LoadGateway()
	if err != nil || strings.Join(gateway.Scopes, ",") != "plots:read,plots:write" ||
		gateway.TokenURL != "http://keycloak:8080/realms/land-plot/protocol/openid-connect/token" {
		t.Fatalf("LoadGateway() = %#v, %v", gateway, err)
	}
}

func TestLoadGatewayTrimsTrailingSlash(t *testing.T) {
	setGatewayEnv(t)
	t.Setenv("GATEWAY_URL", "http://api-gateway:8080/")

	gateway, err := LoadGateway()
	if err != nil || gateway.URL != "http://api-gateway:8080" {
		t.Fatalf("LoadGateway() = %q, %v", gateway.URL, err)
	}
}

func TestLoadGatewayRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"AI_CLIENT_SECRET": " ",
		"AI_CLIENT_ID":     " ",
		"AI_SCOPES":        "  ",
		"OIDC_TOKEN_URL":   "keycloak:8080/token",
		"GATEWAY_URL":      "api-gateway:8080",
		"GATEWAY_TIMEOUT":  "0s",
	}
	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			setGatewayEnv(t)
			t.Setenv(key, value)

			if _, err := LoadGateway(); err == nil {
				t.Fatalf("LoadGateway() with %s=%q succeeded", key, value)
			}
		})
	}
}

func TestLoadLLMAllowsEmptyAPIKey(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")

	llm, err := LoadLLM()
	if err != nil {
		t.Fatalf("LoadLLM() error = %v", err)
	}

	if llm.APIKey != "" || llm.Model != defaultLLMModel || llm.MaxOutputTokens != defaultLLMMaxOutputTokens ||
		llm.MinInterval != defaultLLMMinInterval || llm.ReasoningThinkingLevel != defaultLLMThinkingLevel {
		t.Fatalf("LoadLLM() = %#v", llm)
	}
}

func TestLoadLLMReadsSettings(t *testing.T) {
	t.Setenv("LLM_MODEL", "gemini-3.6-flash")
	t.Setenv("LLM_API_KEY", "key")
	t.Setenv("LLM_MAX_OUTPUT_TOKENS", "512")
	t.Setenv("LLM_THINKING_LEVEL", "low")
	t.Setenv("LLM_MIN_INTERVAL", "10s")

	llm, err := LoadLLM()
	if err != nil {
		t.Fatalf("LoadLLM() error = %v", err)
	}

	if llm.Model != "gemini-3.6-flash" || llm.APIKey != "key" || llm.MaxOutputTokens != 512 ||
		llm.ReasoningThinkingLevel != "low" || llm.MinInterval != 10*time.Second {
		t.Fatalf("LoadLLM() = %#v", llm)
	}
}

func TestLoadLLMRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"LLM_MODEL":             " ",
		"LLM_MAX_OUTPUT_TOKENS": "0",
		"LLM_THINKING_LEVEL":    "minimal",
		"LLM_MIN_INTERVAL":      "fast",
		"LLM_REQUEST_TIMEOUT":   "-1s",
	}
	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)

			if _, err := LoadLLM(); err == nil {
				t.Fatalf("LoadLLM() with %s=%q succeeded", key, value)
			}
		})
	}
}

func TestLoadLimitsAppliesDefaultsAndOverrides(t *testing.T) {
	t.Setenv("AGENT_TOKEN_BUDGET", "500")

	limits, err := LoadLimits()
	if err != nil {
		t.Fatalf("LoadLimits() error = %v", err)
	}

	want := Limits{
		TokenBudget:        500,
		MaxSteps:           defaultAgentMaxSteps,
		UserInputMaxChars:  defaultUserInputMaxChars,
		ToolResultMaxBytes: defaultToolResultMaxBytes,
	}
	if limits != want {
		t.Fatalf("LoadLimits() = %#v, want %#v", limits, want)
	}
}

func TestLoadLimitsRejectsNonPositiveValues(t *testing.T) {
	for _, key := range []string{"AGENT_TOKEN_BUDGET", "AGENT_MAX_STEPS", "USER_INPUT_MAX_CHARS", "TOOL_RESULT_MAX_BYTES"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "0")

			if _, err := LoadLimits(); err == nil {
				t.Fatalf("LoadLimits() with %s=0 succeeded", key)
			}
		})
	}
}

func TestLoadMCPDoesNotRequireAccessToken(t *testing.T) {
	t.Setenv("MCP_ACCESS_TOKEN", "")

	mcp, err := LoadMCP()
	if err != nil || mcp.RateLimitPerMinute != defaultMCPRateLimit {
		t.Fatalf("LoadMCP() = %#v, %v", mcp, err)
	}
}

func TestLoadMCPHTTPReadsAccessToken(t *testing.T) {
	token := strings.Repeat("t", minimumMCPAccessToken)
	t.Setenv("MCP_ACCESS_TOKEN", token)

	mcp, err := LoadMCPHTTP()
	if err != nil {
		t.Fatalf("LoadMCPHTTP() error = %v", err)
	}

	if mcp.HTTPAddress != defaultMCPHTTPAddress || mcp.AccessToken != token {
		t.Fatalf("LoadMCPHTTP() = %#v", mcp)
	}
}

func TestLoadMCPHTTPRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"MCP_ACCESS_TOKEN":          "short",
		"MCP_HTTP_ADDR":             ":99999",
		"MCP_RATE_LIMIT_PER_MINUTE": "-5",
	}
	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			t.Setenv("MCP_ACCESS_TOKEN", strings.Repeat("t", minimumMCPAccessToken))
			t.Setenv(key, value)

			if _, err := LoadMCPHTTP(); err == nil {
				t.Fatalf("LoadMCPHTTP() with %s=%q succeeded", key, value)
			}
		})
	}
}

func TestLoadAgentServerUsesCommandDefaults(t *testing.T) {
	server, err := LoadAgentServer(":8091", "http://localhost:8091")
	if err != nil {
		t.Fatalf("LoadAgentServer() error = %v", err)
	}

	if server.HTTPAddress != ":8091" || server.PublicURL != "http://localhost:8091" ||
		server.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("LoadAgentServer() = %#v", server)
	}
}

func TestLoadAgentServerRejectsInvalidPublicURL(t *testing.T) {
	t.Setenv("AGENT_PUBLIC_URL", "producer-agent")

	if _, err := LoadAgentServer(":8091", "http://localhost:8091"); err == nil {
		t.Fatal("LoadAgentServer() with a relative public URL succeeded")
	}
}

func TestLoadProducerReadsSettings(t *testing.T) {
	t.Setenv("PRODUCER_AGENT_URL", "http://producer-agent:8091")
	t.Setenv("A2A_CALL_TIMEOUT", "2m")

	producer, err := LoadProducer()
	if err != nil {
		t.Fatalf("LoadProducer() error = %v", err)
	}

	if producer.URL != "http://producer-agent:8091" || producer.CallTimeout != 2*time.Minute {
		t.Fatalf("LoadProducer() = %#v", producer)
	}
}
