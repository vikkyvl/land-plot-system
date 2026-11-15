package landapi

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	pathLogin     = "/api/v1/auth/login"
	pathRegister  = "/api/v1/auth/register"
	refreshMargin = 30 * time.Second
)

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

	login, err := client.login(requestContext)
	if hasCode(err, CodeLoginRejected) {
		if err := client.register(requestContext); err != nil {
			return "", err
		}

		login, err = client.login(requestContext)
	}

	if err != nil {
		return "", fmt.Errorf(wrapLogin, err)
	}

	client.token, client.expiresAt = login.AccessToken, login.ExpiresAt

	return client.token, nil
}

func (client *Client) forgetToken(token string) {
	client.tokenMutex.Lock()
	defer client.tokenMutex.Unlock()

	if client.token == token {
		client.token = ""
	}
}

func (client *Client) login(requestContext context.Context) (loginResponse, error) {
	var login loginResponse

	err := client.send(
		requestContext,
		request{method: http.MethodPost, path: pathLogin, body: client.credentials},
		"",
		&login,
	)
	if err != nil {
		return loginResponse{}, err
	}

	if login.AccessToken == "" {
		return loginResponse{}, errMissingAccessToken
	}

	return login, nil
}

func (client *Client) register(requestContext context.Context) error {
	err := client.send(
		requestContext,
		request{method: http.MethodPost, path: pathRegister, body: client.credentials},
		"",
		nil,
	)
	if err != nil && !hasCode(err, CodeUserExists) {
		return fmt.Errorf(wrapRegister, err)
	}

	return nil
}
