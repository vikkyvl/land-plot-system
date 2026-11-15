package bootstrap

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"

	"github.com/vleshchenko/land-plot-system/ai/internal/a2aagent"
)

const defaultCallTimeout = 3 * time.Minute

type a2aCallOptions struct {
	agentURL string
	text     string
	timeout  time.Duration
}

func RunA2ACall(runContext context.Context, arguments []string, stdout, stderr io.Writer) int {
	options, err := parseA2ACallOptions(arguments, stderr)
	if err != nil {
		_, _ = fmt.Fprint(stderr, formatA2ACallUsage)
		return ExitUsage
	}

	callContext, cancel := context.WithTimeout(runContext, options.timeout)
	defer cancel()

	httpClient := &http.Client{Timeout: options.timeout}

	card, err := agentcard.NewResolver(httpClient).Resolve(callContext, options.agentURL)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return ExitError
	}

	printCard(stdout, card, options.text)

	client, err := a2aclient.NewFromEndpoints(
		callContext,
		[]*a2a.AgentInterface{a2a.NewAgentInterface(options.agentURL, a2a.TransportProtocolJSONRPC)},
		a2aclient.WithJSONRPCTransport(httpClient),
	)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return ExitError
	}
	defer func() { _ = client.Destroy() }()

	request := &a2a.SendMessageRequest{Message: a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(options.text))}
	finalState := a2a.TaskStateUnspecified

	for event, err := range client.SendStreamingMessage(callContext, request) {
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return ExitError
		}

		if state := printEvent(stdout, event); state != a2a.TaskStateUnspecified {
			finalState = state
		}
	}

	if finalState != a2a.TaskStateCompleted {
		return ExitError
	}

	return ExitOK
}

func parseA2ACallOptions(arguments []string, stderr io.Writer) (a2aCallOptions, error) {
	flags := flag.NewFlagSet("a2a-call", flag.ContinueOnError)
	flags.SetOutput(stderr)

	timeout := flags.Duration("timeout", defaultCallTimeout, "how long to wait for the task")
	if err := flags.Parse(arguments); err != nil {
		return a2aCallOptions{}, err
	}

	if flags.NArg() < 2 || *timeout <= 0 {
		return a2aCallOptions{}, errMissingCallArguments
	}

	agentURL := strings.TrimRight(flags.Arg(0), "/")

	parsed, err := url.Parse(agentURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return a2aCallOptions{}, errInvalidAgentURL
	}

	text := strings.TrimSpace(strings.Join(flags.Args()[1:], " "))
	if text == "" {
		return a2aCallOptions{}, errMissingCallArguments
	}

	return a2aCallOptions{agentURL: agentURL, text: text, timeout: *timeout}, nil
}

func printCard(output io.Writer, card *a2a.AgentCard, text string) {
	skills := make([]string, 0, len(card.Skills))
	for _, skill := range card.Skills {
		skills = append(skills, skill.ID)
	}

	_, _ = fmt.Fprintf(output, formatAgentCard, card.Name, card.Version, strings.Join(skills, ", "), text)
}

func printEvent(output io.Writer, event a2a.Event) a2a.TaskState {
	switch typed := event.(type) {
	case *a2a.Task:
		_, _ = fmt.Fprintf(output, formatTaskEvent, typed.Status.State, typed.ID)
		printArtifacts(output, typed.Artifacts)

		return typed.Status.State
	case *a2a.TaskStatusUpdateEvent:
		_, _ = fmt.Fprintf(output, formatStatusEvent, typed.Status.State, a2aagent.TaskText(typed.Status.Message))
		return typed.Status.State
	case *a2a.TaskArtifactUpdateEvent:
		printArtifacts(output, []*a2a.Artifact{typed.Artifact})
	case *a2a.Message:
		_, _ = fmt.Fprintf(output, formatMessageEvent, a2aagent.TaskText(typed))
	}

	return a2a.TaskStateUnspecified
}

func printArtifacts(output io.Writer, artifacts []*a2a.Artifact) {
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}

		for _, part := range artifact.Parts {
			encoded, _ := json.Marshal(part.Data())
			_, _ = fmt.Fprintf(output, formatArtifactEvent, artifact.Name, encoded)
		}
	}
}
