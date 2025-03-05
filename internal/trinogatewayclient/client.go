// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package trinogatewayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxResponseBodyLogSize = 1024
)

type Backend struct {
	Name         string `json:"name"`
	ProxyTo      string `json:"proxyTo"`
	RoutingGroup string `json:"routingGroup"`
	Active       bool   `json:"active"`
	ExternalUrl  string `json:"externalUrl"`
}

type Auth struct {
	Login    string
	Password string
}

type TrinoGatewayClient interface {
	AddOrUpdateBackend(ctx context.Context, backend *Backend) error
	DeleteBackend(ctx context.Context, name string) error
	GetAllBackends(ctx context.Context) ([]*Backend, error)
}

func NewTrinoGatewayClient(endpoint string, auth *Auth) (TrinoGatewayClient, error) {
	return &trinoGatewayClientHttpImpl{
		auth:       auth,
		endpoint:   endpoint,
		httpclient: http.DefaultClient,
	}, nil
}

type trinoGatewayClientHttpImpl struct {
	httpclient *http.Client
	auth       *Auth
	endpoint   string
}

func (tg *trinoGatewayClientHttpImpl) getFullUrl(subpath string) string {
	return strings.TrimSuffix(tg.endpoint, "/") + subpath
}

func (tg *trinoGatewayClientHttpImpl) addAuth(request *http.Request) {
	if tg.auth != nil {
		request.SetBasicAuth(tg.auth.Login, tg.auth.Password)
	}
}
func (tg *trinoGatewayClientHttpImpl) AddOrUpdateBackend(ctx context.Context, backend *Backend) error {
	requestBody, err := json.Marshal(backend)
	if err != nil {
		return fmt.Errorf("cant marshal backend: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		tg.getFullUrl("/entity?entityType=GATEWAY_BACKEND"),
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return fmt.Errorf("cant create request: %w", err)
	}
	tg.addAuth(request)

	response, err := tg.httpclient.Do(request)
	if err != nil {
		return fmt.Errorf("cant send request: %w", err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)

	if response.StatusCode != 200 {
		return fmt.Errorf(
			"bad http response code: %d, body: %s",
			response.StatusCode,
			responseBody[:min(len(responseBody), maxResponseBodyLogSize)],
		)
	}
	return nil
}

func (tg *trinoGatewayClientHttpImpl) DeleteBackend(ctx context.Context, name string) error {
	request, err := http.NewRequest(
		http.MethodPost,
		tg.getFullUrl("/gateway/backend/modify/delete"),
		strings.NewReader(name),
	)
	if err != nil {
		return fmt.Errorf("cant create request: %w", err)
	}
	tg.addAuth(request)

	response, err := tg.httpclient.Do(request)
	if err != nil {
		return fmt.Errorf("cant send request: %w", err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)

	if response.StatusCode != 200 {
		return fmt.Errorf(
			"bad http response code: %d, body: %s",
			response.StatusCode,
			responseBody[:min(len(responseBody), maxResponseBodyLogSize)],
		)
	}
	return nil
}

func (tg *trinoGatewayClientHttpImpl) GetAllBackends(ctx context.Context) ([]*Backend, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		tg.getFullUrl("/entity/GATEWAY_BACKEND"),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cant create request: %w", err)
	}
	tg.addAuth(request)

	response, err := tg.httpclient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cant send request: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("cant read response body")
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf(
			"bad http response code: %d, body: %s",
			response.StatusCode,
			responseBody[:min(len(responseBody), maxResponseBodyLogSize)],
		)
	}

	allBackends := []*Backend{}
	if err := json.Unmarshal(responseBody, &allBackends); err != nil {
		return nil, fmt.Errorf(
			"cant unmarshal response: %w, body: %s",
			err,
			responseBody[:min(len(responseBody), maxResponseBodyLogSize)],
		)
	}
	return allBackends, nil
}
