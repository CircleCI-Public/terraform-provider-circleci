// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package runner

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"terraform-provider-circleci/internal/circleci/client"
)

// Runner represents a self-hosted runner instance.
type Runner struct {
	Name           string `json:"name,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	IP             string `json:"ip,omitempty"`
	Version        string `json:"version,omitempty"`
	Status         string `json:"status,omitempty"`
	ResourceClass  string `json:"resource_class,omitempty"`
	FirstConnected string `json:"first_connected,omitempty"`
	LastConnected  string `json:"last_connected,omitempty"`
	LastUsed       string `json:"last_used,omitempty"`
}

// ResourceClassItems represents a set of resource classes.
type ResourceClassItems struct {
	Items []ResourceClass `json:"items"`
}

// ResourceClass represents a runner resource class configuration.
type ResourceClass struct {
	Id            string `json:"id"`
	ResourceClass string `json:"resource_class"`
	Description   string `json:"description,omitempty"`
}

// TokenItems represents a set of tokens.
type TokenItems struct {
	Items []Token `json:"items"`
}

// Token represents a runner authentication token.
type Token struct {
	Id            string `json:"id,omitempty"`
	Nickname      string `json:"nickname,omitempty"`
	ResourceClass string `json:"resource_class,omitempty"`
	Token         string `json:"token,omitempty"` // Only returned on create
	CreatedAt     string `json:"created_at,omitempty"`
}

// CreateResourceClassRequest contains the parameters for creating a resource class.
type CreateResourceClassRequest struct {
	OrganizationID string `json:"org_id"`
	ResourceClass  string `json:"resource_class"`
	Description    string `json:"description,omitempty"`
}

// CreateTokenRequest contains the parameters for creating a runner token.
type CreateTokenRequest struct {
	OrganizationID string `json:"org_id"`
	ResourceClass  string `json:"resource_class"`
	Nickname       string `json:"nickname"`
}

// ListRunnersParams contains filtering parameters for listing runners.
type ListRunnersParams struct {
	ResourceClass string
	Namespace     string
	OrgID         string
}

// UnclaimedTaskCount represents the count of unclaimed tasks for a resource class.
type UnclaimedTaskCount struct {
	UnclaimedTaskCount int `json:"unclaimed_task_count"`
}

// RunningTaskCount represents the count of running tasks for a resource class.
type RunningTaskCount struct {
	RunningRunnerTasks int `json:"running_runner_tasks"`
}

// Service provides methods for interacting with the CircleCI Runner Admin API v3.
type Service struct {
	client  *client.Client
	baseURL string
}

// NewService creates a new Service instance that uses the production runner API.
func NewService(c *client.Client) *Service {
	return &Service{
		client:  c,
		baseURL: "https://runner.circleci.com",
	}
}

// NewServiceWithBaseURL creates a new Service instance with a custom base URL.
// This is primarily useful for testing.
func NewServiceWithBaseURL(c *client.Client, baseURL string) *Service {
	return &Service{
		client:  c,
		baseURL: baseURL,
	}
}

// ListRunners returns a list of runners filtered by the provided parameters.
// At least one filter parameter should be provided.
func (s *Service) ListRunners(ctx context.Context, params ListRunnersParams) ([]Runner, error) {
	values := url.Values{}
	if params.ResourceClass != "" {
		values.Add("resource-class", params.ResourceClass)
	}
	if params.Namespace != "" {
		values.Add("namespace", params.Namespace)
	}
	if params.OrgID != "" {
		values.Add("org-id", params.OrgID)
	}

	query := ""
	if len(values) > 0 {
		query = "?" + values.Encode()
	}

	var runners []Runner
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodGet, s.baseURL+"/api/v3/runner"+query, nil, &runners)
	if err != nil {
		return nil, err
	}

	return runners, nil
}

// ListResourceClasses returns a list of resource classes filtered by namespace and/or organization ID.
// At least one filter parameter should be provided.
func (s *Service) ListResourceClasses(ctx context.Context, namespace, orgID string) (*ResourceClassItems, error) {
	values := url.Values{}
	if namespace != "" {
		values.Add("namespace", namespace)
	}
	if orgID != "" {
		values.Add("org-id", orgID)
	}

	query := ""
	if len(values) > 0 {
		query = "?" + values.Encode()
	}

	var resourceClassItems ResourceClassItems
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodGet, s.baseURL+"/api/v3/runner/resource"+query, nil, &resourceClassItems)
	if err != nil {
		return nil, err
	}

	return &resourceClassItems, nil
}

// CreateResourceClass creates a new runner resource class.
func (s *Service) CreateResourceClass(ctx context.Context, req CreateResourceClassRequest) (*ResourceClass, error) {
	var resourceClass ResourceClass
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodPost, s.baseURL+"/api/v3/runner/resource", req, &resourceClass)
	if err != nil {
		return nil, err
	}

	return &resourceClass, nil
}

// DeleteResourceClass deletes a resource class by ID.
// If force is true, the resource class will be deleted even if it has associated tokens.
func (s *Service) DeleteResourceClass(ctx context.Context, id string, force bool) error {
	path := fmt.Sprintf("%s/api/v3/runner/resource/%s", s.baseURL, id)
	if force {
		path += "/force"
	}

	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodDelete, path, nil, nil)
	return err
}

// ListTokens returns a list of tokens for a specific resource class.
func (s *Service) ListTokens(ctx context.Context, resourceClass string) (*TokenItems, error) {
	values := url.Values{}
	if resourceClass != "" {
		values.Add("resource-class", resourceClass)
	}

	query := ""
	if len(values) > 0 {
		query = "?" + values.Encode()
	}

	var tokens TokenItems
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodGet, s.baseURL+"/api/v3/runner/token"+query, nil, &tokens)
	if err != nil {
		return nil, err
	}

	return &tokens, nil
}

// CreateToken creates a new runner token.
// The token value is only returned in the response and cannot be retrieved later.
func (s *Service) CreateToken(ctx context.Context, req CreateTokenRequest) (*Token, error) {
	var token Token
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodPost, s.baseURL+"/api/v3/runner/token", req, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// DeleteToken deletes a runner token by ID.
func (s *Service) DeleteToken(ctx context.Context, id string) error {
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodDelete, fmt.Sprintf("%s/api/v3/runner/token/%s", s.baseURL, id), nil, nil)
	return err
}

// GetUnclaimedTaskCount returns the number of unclaimed tasks for a resource class.
func (s *Service) GetUnclaimedTaskCount(ctx context.Context, resourceClass string) (*UnclaimedTaskCount, error) {
	values := url.Values{}
	if resourceClass != "" {
		values.Add("resource-class", resourceClass)
	}

	query := ""
	if len(values) > 0 {
		query = "?" + values.Encode()
	}

	var count UnclaimedTaskCount
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodGet, s.baseURL+"/api/v3/runner/tasks"+query, nil, &count)
	if err != nil {
		return nil, err
	}

	return &count, nil
}

// GetRunningTaskCount returns the number of running tasks for a resource class.
func (s *Service) GetRunningTaskCount(ctx context.Context, resourceClass string) (*RunningTaskCount, error) {
	values := url.Values{}
	if resourceClass != "" {
		values.Add("resource-class", resourceClass)
	}

	query := ""
	if len(values) > 0 {
		query = "?" + values.Encode()
	}

	var count RunningTaskCount
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodGet, s.baseURL+"/api/v3/runner/tasks/running"+query, nil, &count)
	if err != nil {
		return nil, err
	}

	return &count, nil
}
