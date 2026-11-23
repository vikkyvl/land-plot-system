package a2aagent

import (
	"net/http"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

const (
	cardVersion = "1.0.0"
	healthPath  = "GET /health"
	healthBody  = `{"status":"ok"}`
	rootPath    = "/"
)

var contentModes = []string{"text/plain", "application/json"}

func NewHandler(card *a2a.AgentCard, executor a2asrv.AgentExecutor) http.Handler {
	router := http.NewServeMux()
	router.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
	router.Handle(rootPath, a2asrv.NewJSONRPCHandler(a2asrv.NewHandler(executor)))
	router.HandleFunc(healthPath, func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(healthBody))
	})

	return router
}

func newCard(name, description, publicURL string, skills []a2a.AgentSkill) *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:                name,
		Description:         description,
		Version:             cardVersion,
		SupportedInterfaces: []*a2a.AgentInterface{a2a.NewAgentInterface(publicURL, a2a.TransportProtocolJSONRPC)},
		DefaultInputModes:   contentModes,
		DefaultOutputModes:  contentModes,
		Capabilities:        a2a.AgentCapabilities{Streaming: true},
		Skills:              skills,
	}
}
