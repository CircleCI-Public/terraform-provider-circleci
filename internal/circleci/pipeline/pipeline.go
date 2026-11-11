// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

type Pipeline struct {
	ID             string                `json:"id,omitempty"`
	Name           string                `json:"name,omitempty"`
	Description    string                `json:"description,omitempty"`
	CreatedAt      string                `json:"created_at,omitempty"`
	ConfigSource   common.ConfigSource   `json:"config_source,omitzero"`
	CheckoutSource common.CheckoutSource `json:"checkout_source,omitzero"`
}

type PipelineItems struct {
	Items []Pipeline `json:"items"`
}

type PipelineService struct {
	client *client.Client
}

func NewPipelineService(c *client.Client) *PipelineService {
	return &PipelineService{client: c}
}

// Get reads a pipeline through v2, then re-reads it through v3 when it is a
// Forgejo pipeline, because v2 reports Forgejo pipelines without their
// repositories.
func (s *PipelineService) Get(ctx context.Context, projectID, pipelineID string) (_ *Pipeline, err error) {
	var pipeline Pipeline
	_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/projects/%s/pipeline-definitions/%s", projectID, pipelineID), nil, &pipeline)
	if err != nil {
		return nil, err
	}
	if !pipeline.isForgejo() {
		return &pipeline, nil
	}

	var resp client.V3Data[v3Pipeline]
	_, err = s.client.V3(ctx, http.MethodGet, "/pipelines/"+url.PathEscape(pipelineID), url.Values{"filter[project_id]": {projectID}}, nil, &resp)
	if err != nil {
		return nil, err
	}
	return fromV3(resp.Data), nil
}

func (s *PipelineService) List(ctx context.Context, projectID string) (_ []Pipeline, err error) {
	var pipelineItems PipelineItems
	_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/projects/%s/pipeline-definitions", projectID), nil, &pipelineItems)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(pipelineItems.Items, Pipeline.isForgejo) {
		return pipelineItems.Items, nil
	}

	v3Items, err := client.V3ListAll[v3Pipeline](ctx, s.client, "/pipelines", url.Values{"filter[project_id]": {projectID}})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]v3Pipeline, len(v3Items))
	for _, item := range v3Items {
		byID[item.ID] = item
	}
	for i, p := range pipelineItems.Items {
		if item, ok := byID[p.ID]; ok && p.isForgejo() {
			pipelineItems.Items[i] = *fromV3(item)
		}
	}
	return pipelineItems.Items, nil
}

// Create creates a Forgejo pipeline through v3 and any other pipeline through
// v2, which does not support Forgejo.
func (s *PipelineService) Create(ctx context.Context, newPipeline Pipeline, projectID string) (_ *Pipeline, err error) {
	if newPipeline.isForgejo() {
		return s.createV3(ctx, newPipeline, projectID)
	}

	var pipeline Pipeline
	_, err = s.client.RequestHelper(ctx, http.MethodPost, fmt.Sprintf("/projects/%s/pipeline-definitions", projectID), newPipeline, &pipeline)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}

func (s *PipelineService) Delete(ctx context.Context, projectID, pipelineID string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, fmt.Sprintf("/projects/%s/pipeline-definitions/%s", projectID, pipelineID), nil, nil)
	return err
}

// Update - The new pipeline param can only have the eseential values:
// name, description, config_source.file_path, checkout_source.provider, checkout_source.repo.external_id
// These are the only values that can be updated with this method, and the object passed can only have these defined.
// A Forgejo pipeline, identified by checkout_source.provider, is updated through v3, which only changes
// name, description and config_source.file_path.
func (s *PipelineService) Update(ctx context.Context, newPipeline Pipeline, projectID, pipelineID string) (_ *Pipeline, err error) {
	if newPipeline.CheckoutSource.Provider == ProviderForgejo {
		return s.updateV3(ctx, newPipeline, projectID, pipelineID)
	}

	var pipeline Pipeline
	_, err = s.client.RequestHelper(ctx, http.MethodPatch, fmt.Sprintf("/projects/%s/pipeline-definitions/%s", projectID, pipelineID), newPipeline, &pipeline)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}
