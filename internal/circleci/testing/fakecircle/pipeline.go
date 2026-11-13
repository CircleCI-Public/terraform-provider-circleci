// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type pipelineDef struct {
	ID                   uuid.UUID
	ProjectID            uuid.UUID
	Name                 string
	Description          string
	CreatedAt            time.Time
	Provider             string
	FilePath             string
	ConfigRepoID         string
	ConfigRepoFullName   string
	CheckoutRepoID       string
	CheckoutRepoFullName string
}

func (s *Service) setupPipelineRoutes(r chi.Router) {
	r.Post("/api/v2/projects/{project-id}/pipeline-definitions", s.postPipelineV2)
	r.Get("/api/v2/projects/{project-id}/pipeline-definitions", s.listPipelinesV2)
	r.Get("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}", s.getPipelineV2)
	r.Patch("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}", s.patchPipelineV2)
	r.Delete("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}", s.deletePipelineV2)
}

type pipelineRepoV2 struct {
	FullName   string `json:"full_name,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
}

type pipelineSourceV2 struct {
	Provider string          `json:"provider,omitempty"`
	Repo     *pipelineRepoV2 `json:"repo,omitempty"`
	FilePath string          `json:"file_path,omitempty"`
}

type pipelineV2 struct {
	ID             uuid.UUID        `json:"id"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	CreatedAt      time.Time        `json:"created_at"`
	ConfigSource   pipelineSourceV2 `json:"config_source"`
	CheckoutSource pipelineSourceV2 `json:"checkout_source"`
}

type pipelineRequestV2 struct {
	Name           *string           `json:"name"`
	Description    *string           `json:"description"`
	ConfigSource   *pipelineSourceV2 `json:"config_source"`
	CheckoutSource *pipelineSourceV2 `json:"checkout_source"`
}

func isGithubProvider(provider string) bool {
	return provider == "github_app" || provider == "github_server"
}

// v2 mirrors the real v2 endpoints, which only know GitHub providers and
// report other providers' pipelines without their repositories.
func (p *pipelineDef) v2() pipelineV2 {
	resp := pipelineV2{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		CreatedAt:    p.CreatedAt,
		ConfigSource: pipelineSourceV2{Provider: p.Provider, FilePath: p.FilePath},
		CheckoutSource: pipelineSourceV2{
			Provider: p.Provider,
		},
	}
	if isGithubProvider(p.Provider) {
		resp.ConfigSource.Repo = &pipelineRepoV2{FullName: p.ConfigRepoFullName, ExternalID: p.ConfigRepoID}
		resp.CheckoutSource.Repo = &pipelineRepoV2{FullName: p.CheckoutRepoFullName, ExternalID: p.CheckoutRepoID}
	}
	return resp
}

func fakeGithubRepoFullName(id string) string {
	return fmt.Sprintf("fake-org/repo-%s", id)
}

// pipelineHasTriggersLocked reports whether any trigger belongs to the
// pipeline, which stops the real API deleting the pipeline or changing its
// checkout repository.
func (s *Service) pipelineHasTriggersLocked(id uuid.UUID) bool {
	for _, t := range s.triggers {
		if t.PipelineID == id.String() {
			return true
		}
	}
	return false
}

func (s *Service) postPipelineV2(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project-id"))
	if badRequest(w, r, "Invalid project ID.", err) {
		return
	}
	var req pipelineRequestV2
	if badRequest(w, r, "Invalid request body.", render.DecodeJSON(r.Body, &req)) {
		return
	}
	if req.Name == nil || req.ConfigSource == nil || req.CheckoutSource == nil ||
		req.ConfigSource.Repo == nil || req.CheckoutSource.Repo == nil {
		msg(w, r, http.StatusBadRequest, "Missing required field.")
		return
	}
	if !isGithubProvider(req.ConfigSource.Provider) || !isGithubProvider(req.CheckoutSource.Provider) {
		msg(w, r, http.StatusBadRequest, "Unsupported provider.")
		return
	}

	p := &pipelineDef{
		ID:                   uuid.New(),
		ProjectID:            projectID,
		Name:                 *req.Name,
		CreatedAt:            time.Now().UTC(),
		Provider:             req.ConfigSource.Provider,
		FilePath:             req.ConfigSource.FilePath,
		ConfigRepoID:         req.ConfigSource.Repo.ExternalID,
		ConfigRepoFullName:   fakeGithubRepoFullName(req.ConfigSource.Repo.ExternalID),
		CheckoutRepoID:       req.CheckoutSource.Repo.ExternalID,
		CheckoutRepoFullName: fakeGithubRepoFullName(req.CheckoutSource.Repo.ExternalID),
	}
	if req.Description != nil {
		p.Description = *req.Description
	}

	s.mu.Lock()
	s.pipelines[p.ID] = p
	s.mu.Unlock()

	respond(w, r, http.StatusCreated, p.v2())
}

func (s *Service) listPipelinesV2(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project-id"))
	if badRequest(w, r, "Invalid project ID.", err) {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]pipelineV2, 0)
	for _, p := range s.pipelines {
		if p.ProjectID == projectID {
			items = append(items, p.v2())
		}
	}
	respond(w, r, http.StatusOK, newListResponse(items))
}

func (s *Service) getPipelineV2(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.lookupPipelineV2Locked(w, r)
	if !ok {
		return
	}
	respond(w, r, http.StatusOK, p.v2())
}

func (s *Service) patchPipelineV2(w http.ResponseWriter, r *http.Request) {
	var req pipelineRequestV2
	if badRequest(w, r, "Invalid request body.", decodeStrict(r, &req)) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.lookupPipelineV2Locked(w, r)
	if !ok {
		return
	}
	if !isGithubProvider(p.Provider) {
		msg(w, r, http.StatusBadRequest, "Unsupported provider.")
		return
	}
	changesCheckout := req.CheckoutSource != nil && req.CheckoutSource.Repo != nil &&
		req.CheckoutSource.Repo.ExternalID != "" && req.CheckoutSource.Repo.ExternalID != p.CheckoutRepoID
	if changesCheckout && s.pipelineHasTriggersLocked(p.ID) {
		msg(w, r, http.StatusBadRequest,
			"the checkout source cannot be edited because this config source has one or more triggers associated with it")
		return
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.ConfigSource != nil && req.ConfigSource.FilePath != "" {
		p.FilePath = req.ConfigSource.FilePath
	}
	if req.CheckoutSource != nil && req.CheckoutSource.Repo != nil && req.CheckoutSource.Repo.ExternalID != "" {
		p.CheckoutRepoID = req.CheckoutSource.Repo.ExternalID
		p.CheckoutRepoFullName = fakeGithubRepoFullName(p.CheckoutRepoID)
	}
	respond(w, r, http.StatusOK, p.v2())
}

func (s *Service) deletePipelineV2(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.lookupPipelineV2Locked(w, r)
	if !ok {
		return
	}
	if s.pipelineHasTriggersLocked(p.ID) {
		msg(w, r, http.StatusBadRequest, "the pipeline cannot be deleted because it has one or more triggers associated with it")
		return
	}
	s.deletePipelineLocked(p.ID)
	msg(w, r, http.StatusOK, "Pipeline definition deleted.")
}

func (s *Service) lookupPipelineV2Locked(w http.ResponseWriter, r *http.Request) (*pipelineDef, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "pipeline-definition-id"))
	if err != nil {
		msg(w, r, http.StatusBadRequest, "Invalid pipeline definition ID.")
		return nil, false
	}
	p, ok := s.pipelines[id]
	if !ok || p.ProjectID.String() != chi.URLParam(r, "project-id") {
		msg(w, r, http.StatusNotFound, "Pipeline definition not found.")
		return nil, false
	}
	return p, true
}

// deletePipelineLocked removes a pipeline and any triggers it still has.
func (s *Service) deletePipelineLocked(id uuid.UUID) {
	delete(s.pipelines, id)
	for tid, t := range s.triggers {
		if t.PipelineID == id.String() {
			delete(s.triggers, tid)
		}
	}
}
