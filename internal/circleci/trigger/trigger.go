// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package trigger

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

type Trigger struct {
	ID          string             `json:"id,omitempty"`
	CreatedAt   string             `json:"created_at,omitempty"`
	CheckoutRef string             `json:"checkout_ref,omitempty"`
	ConfigRef   string             `json:"config_ref,omitempty"`
	EventSource common.EventSource `json:"event_source,omitzero"`
	EventName   string             `json:"event_name,omitempty"`
	EventPreset string             `json:"event_preset,omitempty"`
	Disabled    *bool              `json:"disabled,omitempty"`
	Parameters  map[string]any     `json:"parameters,omitempty"`
}

type TriggerResponse struct {
	ID          string                     `json:"id,omitempty"`
	CreatedAt   string                     `json:"created_at,omitempty"`
	CheckoutRef string                     `json:"checkout_ref,omitempty"`
	ConfigRef   string                     `json:"config_ref,omitempty"`
	EventSource common.EventSourceResponse `json:"event_source,omitzero"`
	EventName   string                     `json:"event_name,omitempty"`
	EventPreset string                     `json:"event_preset,omitempty"`
	Disabled    *bool                      `json:"disabled,omitempty"`
	Parameters  map[string]any             `json:"parameters,omitempty"`

	// PipelineID is the pipeline the trigger belongs to. Only V3 reports it,
	// so it is empty for triggers read through v2.
	PipelineID string `json:"-"`
}

type TriggerItems struct {
	Items []Trigger `json:"items"`
}

type TriggerResponseItems struct {
	Items []TriggerResponse `json:"items"`
}

type TriggerService struct {
	client *client.Client
}

func NewTriggerService(c *client.Client) *TriggerService {
	return &TriggerService{client: c}
}

// Get reads a trigger through v2, then re-reads it through v3 when it is a
// Forgejo trigger, because v2 reports Forgejo triggers without their
// repository.
func (s *TriggerService) Get(ctx context.Context, projectID, triggerID string) (_ *TriggerResponse, err error) {
	var trigger TriggerResponse
	_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/projects/%s/triggers/%s", projectID, triggerID), nil, &trigger)
	if err != nil {
		return nil, err
	}
	if trigger.EventSource.Provider != ProviderForgejo {
		return &trigger, nil
	}

	var resp client.V3Data[v3Trigger]
	_, err = s.client.V3(ctx, http.MethodGet, "/triggers/"+url.PathEscape(triggerID), url.Values{"filter[project_id]": {projectID}}, nil, &resp)
	if err != nil {
		return nil, err
	}
	return fromV3(resp.Data), nil
}

func (s *TriggerService) List(ctx context.Context, projectID, pipelineID string) (_ []TriggerResponse, err error) {
	var triggerResponseItems TriggerResponseItems
	_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/projects/%s/pipeline-definitions/%s/triggers", projectID, pipelineID), nil, &triggerResponseItems)
	if err != nil {
		return nil, err
	}
	isForgejo := func(t TriggerResponse) bool { return t.EventSource.Provider == ProviderForgejo }
	if !slices.ContainsFunc(triggerResponseItems.Items, isForgejo) {
		return triggerResponseItems.Items, nil
	}

	v3Items, err := client.V3ListAll[v3Trigger](ctx, s.client, "/triggers", url.Values{"filter[project_id]": {projectID}})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]v3Trigger, len(v3Items))
	for _, item := range v3Items {
		byID[item.ID] = item
	}
	for i, t := range triggerResponseItems.Items {
		if item, ok := byID[t.ID]; ok && isForgejo(t) {
			triggerResponseItems.Items[i] = *fromV3(item)
		}
	}
	return triggerResponseItems.Items, nil
}

// Create creates a Forgejo trigger through v3 and any other trigger through
// v2, which does not support Forgejo.
func (s *TriggerService) Create(ctx context.Context, newTrigger Trigger, projectID, pipelineID string) (_ *TriggerResponse, err error) {
	if newTrigger.EventSource.Provider == ProviderForgejo {
		return s.createV3(ctx, newTrigger, projectID, pipelineID)
	}

	var triggerResponse TriggerResponse
	_, err = s.client.RequestHelper(ctx, http.MethodPost, fmt.Sprintf("/projects/%s/pipeline-definitions/%s/triggers", projectID, pipelineID), newTrigger, &triggerResponse)
	if err != nil {
		return nil, err
	}

	return &triggerResponse, nil
}

func (s *TriggerService) Delete(ctx context.Context, projectID, triggerID string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, fmt.Sprintf("/projects/%s/triggers/%s", projectID, triggerID), nil, nil)
	return err
}

// Update The new trigger param can only have the esseential values:
// name, description, event_preset, checkout_ref, config_ref, disabled
// These are the only values that can be updated with this method.
// Setting event_source.provider to forgejo, which the v2 update rejects,
// sends the update through v3 instead.
func (s *TriggerService) Update(ctx context.Context, newTrigger Trigger, projectID, triggerID string) (_ *TriggerResponse, err error) {
	if newTrigger.EventSource.Provider == ProviderForgejo {
		return s.updateV3(ctx, newTrigger, projectID, triggerID)
	}

	var trigger TriggerResponse
	_, err = s.client.RequestHelper(ctx, http.MethodPatch, fmt.Sprintf("/projects/%s/triggers/%s", projectID, triggerID), newTrigger, &trigger)
	if err != nil {
		return nil, err
	}

	return &trigger, nil
}
