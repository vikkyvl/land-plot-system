package bootstrap

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLoadAgentSettingsUsesCommandDefaults(t *testing.T) {
	t.Setenv("AI_USER_EMAIL", "agent@example.com")
	t.Setenv("AI_USER_PASSWORD", "AgentPassword123")
	t.Setenv("LLM_API_KEY", "")

	settings, err := loadAgentSettings(producerAddress, producerPublicURL)
	if err != nil {
		t.Fatalf("loadAgentSettings() error = %v", err)
	}

	if settings.server.HTTPAddress != producerAddress || settings.server.PublicURL != producerPublicURL ||
		settings.llm.APIKey != "" {
		t.Fatalf("settings = %#v", settings.server)
	}
}

func TestRunProducerAgentStopsOnInvalidConfiguration(t *testing.T) {
	t.Setenv("AI_USER_EMAIL", "")

	var stderr bytes.Buffer

	if code := RunProducerAgent(context.Background(), &stderr); code != ExitError ||
		!strings.Contains(stderr.String(), "AI_USER_EMAIL must not be empty") {
		t.Fatalf("RunProducerAgent() = %d, stderr = %q", code, stderr.String())
	}
}
