// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"context"
	"fmt"
	"net/http"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

type Webhook struct {
	Id            string       `json:"id,omitempty"`
	Name          string       `json:"name,omitempty"`
	Url           string       `json:"url,omitempty"`
	VerifyTls     *bool        `json:"verify-tls,omitempty"`
	SigningSecret string       `json:"signing-secret,omitempty"`
	UpdatedAt     string       `json:"updated_at,omitempty"`
	CreatedAt     string       `json:"created_at,omitempty"`
	Scope         common.Scope `json:"scope,omitempty"`
	Events        []string     `json:"events,omitempty"`
}

type WebhookService struct {
	client *client.Client
}

func NewWebhookService(c *client.Client) *WebhookService {
	return &WebhookService{client: c}
}

func (s *WebhookService) Get(ctx context.Context, webhookID string) (_ *Webhook, err error) {
	var webhook Webhook
	_, err = s.client.RequestHelper(ctx, http.MethodGet, "/webhook/"+webhookID, nil, &webhook)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (s *WebhookService) List(ctx context.Context, scopeID string) (_ []Webhook, err error) {
	var nextPageToken string
	var webhookList []Webhook
	for {
		var response common.PaginatedResponse[Webhook]
		_, err = s.client.RequestHelper(ctx, http.MethodGet,
			fmt.Sprintf("/webhook?scope-id=%s&scope-type=project&page-token=%s", scopeID, nextPageToken),
			nil,
			&response,
		)
		if err != nil {
			return nil, err
		}

		webhookList = append(webhookList, response.Items...)
		if response.NextPageToken == "" {
			break
		}
		nextPageToken = response.NextPageToken
	}
	return webhookList, nil
}

func (s *WebhookService) Create(ctx context.Context, newWebhook Webhook) (_ *Webhook, err error) {
	var webhook Webhook
	_, err = s.client.RequestHelper(ctx, http.MethodPost, "/webhook", newWebhook, &webhook)
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

// Update - The scope cannot be updated.
func (s *WebhookService) Update(ctx context.Context, newWebhook Webhook, webhookID string) (_ *Webhook, err error) {
	var webhook Webhook
	_, err = s.client.RequestHelper(ctx, http.MethodPut, "/webhook/"+webhookID, newWebhook, &webhook)
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (s *WebhookService) Delete(ctx context.Context, webhookID string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, "/webhook/"+webhookID, nil, nil)
	return err
}
