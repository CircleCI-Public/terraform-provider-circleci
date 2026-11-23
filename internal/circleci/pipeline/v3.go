// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package pipeline

import (
	"context"
	"net/http"
	"net/url"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

// ProviderForgejo is the provider whose pipelines only the V3 API supports.
const ProviderForgejo = "forgejo"

func (p Pipeline) isForgejo() bool {
	return p.ConfigSource.Provider == ProviderForgejo || p.CheckoutSource.Provider == ProviderForgejo
}

type v3VCS struct {
	Provider     string `json:"provider"`
	RepoID       string `json:"repo_id"`
	RepoFullName string `json:"repo_full_name"`
}

func newV3VCS(provider string, repo common.Repo) *v3VCS {
	return &v3VCS{Provider: provider, RepoID: repo.ExternalId, RepoFullName: repo.FullName}
}

func (v *v3VCS) repo() common.Repo {
	if v == nil {
		return common.Repo{}
	}
	return common.Repo{FullName: v.RepoFullName, ExternalId: v.RepoID}
}

type v3PipelineAttributes struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	Checkout    struct {
		VCS *v3VCS `json:"vcs"`
	} `json:"checkout"`
	Config struct {
		FilePath string `json:"file_path"`
		VCS      *v3VCS `json:"vcs"`
	} `json:"config"`
}

type v3PipelineReferences struct {
	Project client.V3Ref `json:"project"`
}

type v3Pipeline = client.V3Entity[v3PipelineAttributes, v3PipelineReferences]

func (attrs v3PipelineAttributes) pipeline(id string) *Pipeline {
	p := &Pipeline{
		ID:          id,
		Name:        attrs.Name,
		Description: attrs.Description,
		CreatedAt:   attrs.CreatedAt,
		ConfigSource: common.ConfigSource{
			Repo:     attrs.Config.VCS.repo(),
			FilePath: attrs.Config.FilePath,
		},
		CheckoutSource: common.CheckoutSource{
			Repo: attrs.Checkout.VCS.repo(),
		},
	}
	if attrs.Config.VCS != nil {
		p.ConfigSource.Provider = attrs.Config.VCS.Provider
	}
	if attrs.Checkout.VCS != nil {
		p.CheckoutSource.Provider = attrs.Checkout.VCS.Provider
	}
	return p
}

func (s *PipelineService) createV3(ctx context.Context, p Pipeline, projectID string) (*Pipeline, error) {
	type config struct {
		Type     string `json:"type"`
		FilePath string `json:"file_path"`
		VCS      *v3VCS `json:"vcs"`
	}
	type checkout struct {
		VCS *v3VCS `json:"vcs"`
	}
	type attributes struct {
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Config      config   `json:"config"`
		Checkout    checkout `json:"checkout"`
	}
	req := client.V3Data[client.V3Entity[attributes, v3PipelineReferences]]{
		Data: client.V3Entity[attributes, v3PipelineReferences]{
			Attributes: attributes{
				Name:        p.Name,
				Description: p.Description,
				Config: config{
					Type:     "vcs",
					FilePath: p.ConfigSource.FilePath,
					VCS:      newV3VCS(p.ConfigSource.Provider, p.ConfigSource.Repo),
				},
				Checkout: checkout{VCS: newV3VCS(p.CheckoutSource.Provider, p.CheckoutSource.Repo)},
			},
			References: v3PipelineReferences{Project: client.V3Ref{ID: projectID}},
		},
	}

	var resp client.V3Data[v3Pipeline]
	if _, err := s.client.V3(ctx, http.MethodPost, "/pipelines", nil, req, &resp); err != nil {
		return nil, err
	}
	return fromV3(resp.Data), nil
}

func (s *PipelineService) updateV3(ctx context.Context, p Pipeline, projectID, pipelineID string) (*Pipeline, error) {
	type config struct {
		FilePath string `json:"file_path"`
	}
	req := struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Config      *config `json:"config,omitempty"`
	}{Name: p.Name, Description: p.Description}
	if p.ConfigSource.FilePath != "" {
		req.Config = &config{FilePath: p.ConfigSource.FilePath}
	}

	var resp client.V3Data[v3Pipeline]
	_, err := s.client.V3(ctx, http.MethodPost, "/pipelines/"+url.PathEscape(pipelineID)+"/update",
		url.Values{"filter[project_id]": {projectID}}, req, &resp)
	if err != nil {
		return nil, err
	}
	return fromV3(resp.Data), nil
}

func fromV3(e v3Pipeline) *Pipeline {
	return e.Attributes.pipeline(e.ID)
}
