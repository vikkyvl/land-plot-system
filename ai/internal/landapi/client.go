package landapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

const (
	maximumResponseBytes = 1 << 20
	contentTypeJSON      = "application/json"
	bearerPrefix         = "Bearer "
)

type Config struct {
	BaseURL     string
	Timeout     time.Duration
	Credentials ClientCredentials
}

type Client struct {
	baseURL     string
	httpClient  *http.Client
	credentials ClientCredentials
	now         func() time.Time

	tokenMutex sync.Mutex
	token      string
	expiresAt  time.Time
}

type request struct {
	method        string
	path          string
	body          any
	authenticated bool
}

func New(clientConfig Config) *Client {
	return &Client{
		baseURL:     clientConfig.BaseURL,
		httpClient:  &http.Client{Timeout: clientConfig.Timeout},
		credentials: clientConfig.Credentials,
		now:         time.Now,
	}
}

func (client *Client) call(requestContext context.Context, apiRequest request, result any) error {
	if !apiRequest.authenticated {
		return client.send(requestContext, apiRequest, "", result)
	}

	token, err := client.accessToken(requestContext)
	if err != nil {
		return err
	}

	err = client.send(requestContext, apiRequest, token, result)
	if !isUnauthorized(err) {
		return err
	}

	client.forgetToken(token)

	if token, err = client.accessToken(requestContext); err != nil {
		return err
	}

	return client.send(requestContext, apiRequest, token, result)
}

func (client *Client) send(requestContext context.Context, apiRequest request, token string, result any) error {
	httpRequest, err := client.newHTTPRequest(requestContext, apiRequest, token)
	if err != nil {
		return err
	}

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf(wrapSendRequest, apiRequest.method, apiRequest.path, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes))
	if err != nil {
		return fmt.Errorf(wrapReadResponse, apiRequest.method, apiRequest.path, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(response.StatusCode, body)
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf(wrapDecodeResponse, apiRequest.method, apiRequest.path, err)
	}

	return nil
}

func (client *Client) newHTTPRequest(
	requestContext context.Context,
	apiRequest request,
	token string,
) (*http.Request, error) {
	var body io.Reader

	if apiRequest.body != nil {
		encoded, err := json.Marshal(apiRequest.body)
		if err != nil {
			return nil, fmt.Errorf(wrapEncodeRequest, apiRequest.method, apiRequest.path, err)
		}

		body = bytes.NewReader(encoded)
	}

	httpRequest, err := http.NewRequestWithContext(
		requestContext,
		apiRequest.method,
		client.baseURL+apiRequest.path,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf(wrapBuildRequest, apiRequest.method, apiRequest.path, err)
	}

	httpRequest.Header.Set("Accept", contentTypeJSON)

	if body != nil {
		httpRequest.Header.Set("Content-Type", contentTypeJSON)
	}

	if token != "" {
		httpRequest.Header.Set("Authorization", bearerPrefix+token)
	}

	if requestID := logging.RequestID(requestContext); requestID != "" {
		httpRequest.Header.Set(logging.HeaderRequestID, requestID)
	}

	return httpRequest, nil
}

func newAPIError(status int, body []byte) *APIError {
	var decoded errorResponse
	if err := json.Unmarshal(body, &decoded); err != nil || decoded.Code == "" {
		return &APIError{Status: status, Message: unexpectedStatusText}
	}

	return &APIError{Status: status, Code: decoded.Code, Message: decoded.Message}
}
