// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package envproject

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

type EnvVariable struct {
	Value     string    `json:"value,omitempty"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created-at,omitempty"`
}

type EnvService struct {
	client *client.Client
}

func NewEnvService(c *client.Client) *EnvService {
	return &EnvService{client: c}
}

func (s *EnvService) List(ctx context.Context, projectSlug string) (_ []EnvVariable, err error) {
	var nextPageToken string
	var contextList []EnvVariable
	for {
		var response common.PaginatedResponse[EnvVariable]
		_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/project/%s/envvar?page-token=%s", projectSlug, nextPageToken), nil, &response)
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

func (s *EnvService) Get(ctx context.Context, projectSlug, name string) (_ *EnvVariable, err error) {
	var envVariable EnvVariable
	_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/project/%s/envvar/%s", projectSlug, name), nil, &envVariable)
	if err != nil {
		return nil, err
	}
	return &envVariable, nil
}

func (s *EnvService) Create(ctx context.Context, projectSlug, value, name string) (_ *EnvVariable, err error) {
	payload := map[string]string{
		"value": value,
		"name":  name,
	}
	var envVariable EnvVariable
	_, err = s.client.RequestHelper(ctx, http.MethodPost, fmt.Sprintf("/project/%s/envvar", projectSlug), payload, &envVariable)
	if err != nil {
		return nil, err
	}
	return &envVariable, nil
}

func (s *EnvService) Delete(ctx context.Context, projectSlug, name string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, fmt.Sprintf("/project/%s/envvar/%s", projectSlug, name), nil, nil)
	return err
}
