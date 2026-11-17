package bootstrap

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestParseMCPTransport(t *testing.T) {
	tests := map[string]struct {
		arguments []string
		want      string
		valid     bool
	}{
		"default":         {nil, transportStdio, true},
		"http":            {[]string{"-transport", "http"}, transportHTTP, true},
		"unknown":         {[]string{"-transport", "sse"}, "", false},
		"extra arguments": {[]string{"-transport", "stdio", "extra"}, "", false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			transport, err := parseMCPTransport(test.arguments, io.Discard)
			if (err == nil) != test.valid || transport != test.want {
				t.Fatalf("parseMCPTransport() = %q, %v", transport, err)
			}
		})
	}
}

func TestRunMCPServerPrintsUsageForUnknownTransport(t *testing.T) {
	var stderr bytes.Buffer

	code := RunMCPServer(context.Background(), []string{"-transport", "sse"}, &stderr)
	if code != ExitUsage || !strings.Contains(stderr.String(), "usage: mcp-server") {
		t.Fatalf("RunMCPServer() = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunMCPServerOverHTTPRequiresAnAccessToken(t *testing.T) {
	t.Setenv("AI_CLIENT_SECRET", "ai-client-secret")
	t.Setenv("GATEWAY_URL", "http://127.0.0.1:1")
	t.Setenv("MCP_ACCESS_TOKEN", "")

	var stderr bytes.Buffer

	code := RunMCPServer(context.Background(), []string{"-transport", "http"}, &stderr)
	if code != ExitError || !strings.Contains(stderr.String(), "MCP_ACCESS_TOKEN must not be empty") {
		t.Fatalf("RunMCPServer() = %d, stderr = %q", code, stderr.String())
	}
}
