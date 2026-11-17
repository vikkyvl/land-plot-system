package landapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	refreshMargin          = 30 * time.Second
	grantClientCredentials = "client_credentials"
	contentTypeForm        = "application/x-www-form-urlencoded"
	scopeSeparator         = " "
)

type ClientCredentials struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

func (client *Client) Authenticate(requestContext context.Context) error {
	_, err := client.accessToken(requestContext)
	return err
}

func (client *Client) accessToken(requestContext context.Context) (string, error) {
	client.tokenMutex.Lock()
	defer client.tokenMutex.Unlock()

	if client.token != "" && client.now().Add(refreshMargin).Before(client.expiresAt) {
		return client.token, nil
	}

	issued, err := client.requestToken(requestContext)
	if err != nil {
		return "", fmt.Errorf(wrapRequestToken, err)
	}

	client.token = issued.AccessToken
	client.expiresAt = client.now().Add(time.Duration(issued.ExpiresIn) * time.Second)

	return client.token, nil
}

func (client *Client) forgetToken(token string) {
	client.tokenMutex.Lock()
	defer client.tokenMutex.Unlock()

	if client.token == token {
		client.token = ""
	}
}

func (client *Client) requestToken(requestContext context.Context) (tokenResponse, error) {
	form := url.Values{
		"grant_type":    {grantClientCredentials},
		"client_id":     {client.credentials.ClientID},
		"client_secret": {client.credentials.ClientSecret},
	}
	if len(client.credentials.Scopes) > 0 {
		form.Set("scope", strings.Join(client.credentials.Scopes, scopeSeparator))
	}

	httpRequest, err := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		client.credentials.TokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return tokenResponse{}, fmt.Errorf(wrapBuildGrantRequest, err)
	}

	httpRequest.Header.Set("Content-Type", contentTypeForm)
	httpRequest.Header.Set("Accept", contentTypeJSON)

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return tokenResponse{}, fmt.Errorf(wrapSendGrantRequest, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes))
	if err != nil {
		return tokenResponse{}, fmt.Errorf(wrapReadGrantResponse, err)
	}

	if response.StatusCode != http.StatusOK {
		return tokenResponse{}, newTokenError(response.StatusCode, body)
	}

	var issued tokenResponse
	if err := json.Unmarshal(body, &issued); err != nil {
		return tokenResponse{}, fmt.Errorf(wrapDecodeGrantResponse, err)
	}

	if issued.AccessToken == "" {
		return tokenResponse{}, errMissingAccessToken
	}

	return issued, nil
}

func newTokenError(status int, body []byte) *APIError {
	var decoded tokenErrorResponse
	if err := json.Unmarshal(body, &decoded); err != nil || decoded.Error == "" {
		return &APIError{Status: status, Message: unexpectedStatusText}
	}

	return &APIError{Status: status, Code: decoded.Error, Message: decoded.ErrorDescription}
}
