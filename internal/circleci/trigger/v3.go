// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package trigger

import (
	"context"
	"net/http"
	"net/url"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

// ProviderForgejo is the provider whose triggers only the V3 API supports.
const ProviderForgejo = "forgejo"

type v3VCS struct {
	Provider     string `json:"provider"`
	RepoID       string `json:"repo_id"`
	RepoFullName string `json:"repo_full_name"`
}

type v3Ref struct {
	Ref *string `json:"ref"`
}

type v3Filter struct {
	Preset string `json:"preset,omitempty"`
}

type v3TriggerAttributes struct {
	IsDisabled bool   `json:"is_disabled"`
	CreatedAt  string `json:"created_at"`
	Checkout   *v3Ref `json:"checkout"`
	Config     *v3Ref `json:"config"`
	Event      struct {
		Type   string    `json:"type"`
		VCS    *v3VCS    `json:"vcs"`
		Filter *v3Filter `json:"filter"`
	} `json:"event"`
}

type v3TriggerReferences struct {
	Project  client.V3Ref `json:"project"`
	Pipeline client.V3Ref `json:"pipeline"`
}

type v3Trigger = client.V3Entity[v3TriggerAttributes, v3TriggerReferences]

func refString(r *v3Ref) string {
	if r == nil || r.Ref == nil {
		return ""
	}
	return *r.Ref
}

// refWrite sets ref, or clears it when ref is empty, which also leaves an
// unset ref unset.
func refWrite(ref string) *v3Ref {
	return &v3Ref{Ref: &ref}
}

func fromV3(e v3Trigger) *TriggerResponse {
	a := e.Attributes
	disabled := a.IsDisabled
	t := &TriggerResponse{
		ID:          e.ID,
		CreatedAt:   a.CreatedAt,
		CheckoutRef: refString(a.Checkout),
		ConfigRef:   refString(a.Config),
		Disabled:    &disabled,
		PipelineID:  e.References.Pipeline.ID,
	}
	if a.Event.VCS != nil {
		t.EventSource = common.EventSourceResponse{
			Provider: a.Event.VCS.Provider,
			Repo:     common.Repo{FullName: a.Event.VCS.RepoFullName, ExternalId: a.Event.VCS.RepoID},
		}
	}
	if a.Event.Filter != nil {
		t.EventPreset = a.Event.Filter.Preset
	}
	return t
}

func (s *TriggerService) createV3(ctx context.Context, t Trigger, projectID, pipelineID string) (*TriggerResponse, error) {
	type event struct {
		Type   string    `json:"type"`
		VCS    v3VCS     `json:"vcs"`
		Filter *v3Filter `json:"filter,omitempty"`
	}
	type attributes struct {
		IsDisabled bool   `json:"is_disabled"`
		Checkout   *v3Ref `json:"checkout,omitempty"`
		Config     *v3Ref `json:"config,omitempty"`
		Event      event  `json:"event"`
	}
	type references struct {
		Pipeline client.V3Ref `json:"pipeline"`
	}

	attrs := attributes{
		Event: event{
			Type: "vcs",
			VCS: v3VCS{
				Provider:     t.EventSource.Provider,
				RepoID:       t.EventSource.Repo.ExternalId,
				RepoFullName: t.EventSource.Repo.FullName,
			},
		},
	}
	if t.Disabled != nil {
		attrs.IsDisabled = *t.Disabled
	}
	if t.CheckoutRef != "" {
		attrs.Checkout = refWrite(t.CheckoutRef)
	}
	if t.ConfigRef != "" {
		attrs.Config = refWrite(t.ConfigRef)
	}
	if t.EventPreset != "" {
		attrs.Event.Filter = &v3Filter{Preset: t.EventPreset}
	}
	req := client.V3Data[client.V3Entity[attributes, references]]{
		Data: client.V3Entity[attributes, references]{
			Attributes: attrs,
			References: references{Pipeline: client.V3Ref{ID: pipelineID}},
		},
	}

	var resp client.V3Data[v3Trigger]
	_, err := s.client.V3(ctx, http.MethodPost, "/triggers", url.Values{"filter[project_id]": {projectID}}, req, &resp)
	if err != nil {
		return nil, err
	}
	return fromV3(resp.Data), nil
}

func (s *TriggerService) updateV3(ctx context.Context, t Trigger, projectID, triggerID string) (*TriggerResponse, error) {
	type event struct {
		Filter *v3Filter `json:"filter,omitempty"`
	}
	req := struct {
		IsDisabled *bool  `json:"is_disabled,omitempty"`
		Checkout   *v3Ref `json:"checkout"`
		Config     *v3Ref `json:"config"`
		Event      *event `json:"event,omitempty"`
	}{
		IsDisabled: t.Disabled,
		Checkout:   refWrite(t.CheckoutRef),
		Config:     refWrite(t.ConfigRef),
	}
	if t.EventPreset != "" {
		req.Event = &event{Filter: &v3Filter{Preset: t.EventPreset}}
	}

	var resp client.V3Data[v3Trigger]
	_, err := s.client.V3(ctx, http.MethodPost, "/triggers/"+url.PathEscape(triggerID)+"/update",
		url.Values{"filter[project_id]": {projectID}}, req, &resp)
	if err != nil {
		return nil, err
	}
	return fromV3(resp.Data), nil
}
