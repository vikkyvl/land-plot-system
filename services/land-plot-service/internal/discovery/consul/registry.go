package consul

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	healthCheckInterval = "5s"
	healthCheckTimeout  = "2s"
	deregisterAfter     = "1m"
	healthPath          = "/health"
	loopbackAddress     = "127.0.0.1"
)

type Registration struct {
	ID                string
	Name              string
	Address           string
	Port              int
	HealthCheckTarget string
}

type registrationPayload struct {
	ID      string       `json:"ID"`
	Name    string       `json:"Name"`
	Address string       `json:"Address"`
	Port    int          `json:"Port"`
	Check   checkPayload `json:"Check"`
}

type checkPayload struct {
	HTTP                           string `json:"HTTP"`
	Interval                       string `json:"Interval"`
	Timeout                        string `json:"Timeout"`
	DeregisterCriticalServiceAfter string `json:"DeregisterCriticalServiceAfter"`
}

type Registry struct {
	baseURL string
	client  *http.Client
}

func NewRegistry(baseURL string, client *http.Client) *Registry {
	return &Registry{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (registry *Registry) Register(requestContext context.Context, registration Registration) error {
	body, err := json.Marshal(registrationPayload{
		ID:      registration.ID,
		Name:    registration.Name,
		Address: registration.Address,
		Port:    registration.Port,
		Check: checkPayload{
			HTTP:                           registration.HealthCheckTarget,
			Interval:                       healthCheckInterval,
			Timeout:                        healthCheckTimeout,
			DeregisterCriticalServiceAfter: deregisterAfter,
		},
	})
	if err != nil {
		return fmt.Errorf(wrapEncodeRegistration, err)
	}

	if err := registry.sendPut(requestContext, "/v1/agent/service/register", bytes.NewReader(body)); err != nil {
		return fmt.Errorf(wrapRegisterService, err)
	}

	return nil
}

func (registry *Registry) Deregister(requestContext context.Context, instanceID string) error {
	if err := registry.sendPut(
		requestContext,
		"/v1/agent/service/deregister/"+url.PathEscape(instanceID),
		nil,
	); err != nil {
		return fmt.Errorf(wrapDeregisterService, err)
	}

	return nil
}

func (registry *Registry) sendPut(requestContext context.Context, path string, body io.Reader) error {
	request, err := http.NewRequestWithContext(requestContext, http.MethodPut, registry.baseURL+path, body)
	if err != nil {
		return fmt.Errorf(wrapCreateRequest, err)
	}

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := registry.client.Do(request)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(formatConsulReturned, response.Status)
	}

	return nil
}

func AdvertiseAddress(requestContext context.Context, configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf(wrapGetHostname, err)
	}

	addresses, err := net.DefaultResolver.LookupIPAddr(requestContext, hostname)
	if err != nil {
		return "", fmt.Errorf(wrapResolveHostname, hostname, err)
	}

	for _, address := range addresses {
		if ipv4 := address.IP.To4(); ipv4 != nil && !ipv4.IsLoopback() {
			return ipv4.String(), nil
		}
	}

	return loopbackAddress, nil
}

func HealthURL(address string, port int) string {
	return "http://" + net.JoinHostPort(address, strconv.Itoa(port)) + healthPath
}
