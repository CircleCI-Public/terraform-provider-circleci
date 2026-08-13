// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package envcontext

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

type EnvVariable struct {
	Variable  string    `json:"variable,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	ContextId string    `json:"context_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type EnvService struct {
	client *client.Client
}

func NewEnvService(c *client.Client) *EnvService {
	return &EnvService{client: c}
}

func (s *EnvService) List(ctx context.Context, contextID string) (_ []EnvVariable, err error) {
	var nextPageToken string
	var contextList []EnvVariable
	for {
		var response common.PaginatedResponse[EnvVariable]
		_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/context/%s/environment-variable?page-token=%s", contextID, nextPageToken), nil, &response)
		if err != nil {
			return nil, err
		}

		contextList = append(contextList, response.Items...)
		if response.NextPageToken == "" {
			break
		}
		nextPageToken = response.NextPageToken
	}
	return contextList, nil
}

func (s *EnvService) Create(ctx context.Context, contextID, value, name string) (_ *EnvVariable, err error) {
	payload := map[string]string{
		"value": value,
	}
	var envVariable EnvVariable
	_, err = s.client.RequestHelper(ctx, http.MethodPut, fmt.Sprintf("/context/%s/environment-variable/%s", contextID, name), payload, &envVariable)
	if err != nil {
		return nil, err
	}
	return &envVariable, nil
}

func (s *EnvService) Delete(ctx context.Context, contextID, name string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, fmt.Sprintf("/context/%s/environment-variable/%s", contextID, name), nil, nil)
	return err
}
