package consul

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery"
)

type healthEntry struct {
	Node struct {
		Address string `json:"Address"`
	} `json:"Node"`
	Service struct {
		ID      string `json:"ID"`
		Address string `json:"Address"`
		Port    int    `json:"Port"`
	} `json:"Service"`
}

type Resolver struct {
	baseURL string
	client  *http.Client
}

func NewResolver(baseURL string, client *http.Client) *Resolver {
	return &Resolver{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (resolver *Resolver) Resolve(requestContext context.Context, serviceName string) ([]discovery.Instance, error) {
	endpoint := resolver.baseURL + "/v1/health/service/" + url.PathEscape(serviceName) + "?passing=true"

	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf(wrapCreateDiscoveryRequest, err)
	}

	response, err := resolver.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf(wrapQueryConsul, err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(formatQueryConsulReturned, response.Status)
	}

	var entries []healthEntry
	if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf(wrapDecodeConsulResponse, err)
	}

	return toInstances(entries), nil
}

func toInstances(entries []healthEntry) []discovery.Instance {
	instances := make([]discovery.Instance, 0, len(entries))
	for _, entry := range entries {
		address := cmp.Or(entry.Service.Address, entry.Node.Address)
		if entry.Service.ID == "" || address == "" || entry.Service.Port < 1 {
			continue
		}

		instances = append(
			instances,
			discovery.Instance{ID: entry.Service.ID, Address: address, Port: entry.Service.Port},
		)
	}

	slices.SortFunc(
		instances,
		func(first, second discovery.Instance) int { return strings.Compare(first.ID, second.ID) },
	)

	return instances
}
