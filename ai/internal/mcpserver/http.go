package mcpserver

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	MCPPath    = "/mcp"
	healthPath = "GET /health"
	healthBody = `{"status":"ok"}`
)

func NewHTTPHandler(server *mcp.Server, accessToken string) http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)

	verifyToken := func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		if subtle.ConstantTimeCompare([]byte(token), []byte(accessToken)) != 1 {
			return nil, auth.ErrInvalidToken
		}

		return &auth.TokenInfo{}, nil
	}

	router := http.NewServeMux()
	router.Handle(MCPPath, auth.RequireBearerToken(
		verifyToken,
		&auth.RequireBearerTokenOptions{AllowMissingExpiration: true},
	)(streamable))
	router.HandleFunc(healthPath, func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", mimeTypeJSON)
		_, _ = response.Write([]byte(healthBody))
	})

	return router
}
