// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

func (s *Service) setupPipelineRoutes(r chi.Router) {
	r.Post("/api/v2/projects/{project-id}/pipeline-definitions", s.postPipelineDefinition)
	r.Get("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}", s.getPipelineDefinition)
	r.Patch("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}", s.patchPipelineDefinition)
	r.Delete("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}", s.deletePipelineDefinition)
}

type pipelineRepo struct {
	FullName   string `json:"full_name,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
}

type pipelineConfigSource struct {
	Provider string       `json:"provider,omitempty"`
	Repo     pipelineRepo `json:"repo"`
	FilePath string       `json:"file_path,omitempty"`
}

type pipelineCheckoutSource struct {
	Provider string       `json:"provider,omitempty"`
	Repo     pipelineRepo `json:"repo"`
}

type pipelineDefinition struct {
	ID             uuid.UUID              `json:"id"`
	ProjectID      string                 `json:"-"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	CreatedAt      time.Time              `json:"created_at"`
	ConfigSource   pipelineConfigSource   `json:"config_source"`
	CheckoutSource pipelineCheckoutSource `json:"checkout_source"`
}

// pipelineDefinitionPatch is the update body. The members are pointers so the
// handler can tell an omitted field from an empty one.
type pipelineDefinitionPatch struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	ConfigSource *struct {
		FilePath *string `json:"file_path"`
	} `json:"config_source"`
	CheckoutSource *struct {
		Provider *string `json:"provider"`
		Repo     *struct {
			ExternalID *string `json:"external_id"`
		} `json:"repo"`
	} `json:"checkout_source"`
}

// PipelineDefinitionUpdates returns the body of every pipeline definition
// update the fake has decoded, oldest first, rejected ones included. Each map
// holds only the fields the request actually carried.
func (s *Service) PipelineDefinitionUpdates() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.pipelineUpdates)
}

func fakeRepoFullName(externalID string) string {
	return fmt.Sprintf("fake-org/repo-%s", externalID)
}

// Handlers below here

func (s *Service) postPipelineDefinition(w http.ResponseWriter, r *http.Request) {
	var req pipelineDefinition
	if badRequest(w, r, "Invalid request body.", render.DecodeJSON(r.Body, &req)) {
		return
	}

	p := &pipelineDefinition{
		ID:          uuid.New(),
		ProjectID:   chi.URLParam(r, "project-id"),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
		ConfigSource: pipelineConfigSource{
			Provider: req.ConfigSource.Provider,
			FilePath: req.ConfigSource.FilePath,
			Repo: pipelineRepo{
				ExternalID: req.ConfigSource.Repo.ExternalID,
				FullName:   fakeRepoFullName(req.ConfigSource.Repo.ExternalID),
			},
		},
		CheckoutSource: pipelineCheckoutSource{
			Provider: req.CheckoutSource.Provider,
			Repo: pipelineRepo{
				ExternalID: req.CheckoutSource.Repo.ExternalID,
				FullName:   fakeRepoFullName(req.CheckoutSource.Repo.ExternalID),
			},
		},
	}

	s.mu.Lock()
	s.pipelines[p.ID] = p
	s.mu.Unlock()

	respond(w, r, http.StatusOK, p)
}

func (s *Service) getPipelineDefinition(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.lookupPipelineLocked(w, r)
	if !ok {
		return
	}

	respond(w, r, http.StatusOK, p)
}

// patchPipelineDefinition rejects any checkout_source while the definition has
// triggers, as the real API does, even when the values are unchanged.
func (s *Service) patchPipelineDefinition(w http.ResponseWriter, r *http.Request) {
	var raw map[string]any
	if badRequest(w, r, "Invalid request body.", render.DecodeJSON(r.Body, &raw)) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pipelineUpdates = append(s.pipelineUpdates, maps.Clone(raw))

	var req pipelineDefinitionPatch
	body, err := json.Marshal(raw)
	if badRequest(w, r, "Invalid request body.", err) {
		return
	}
	if badRequest(w, r, "Invalid request body.", json.Unmarshal(body, &req)) {
		return
	}

	p, ok := s.lookupPipelineLocked(w, r)
	if !ok {
		return
	}

	if req.CheckoutSource != nil && s.pipelineHasTriggersLocked(p.ID) {
		msg(w, r, http.StatusBadRequest, "The checkout source cannot be edited because this pipeline definition has one or more triggers associated with it.")
		return
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.ConfigSource != nil && req.ConfigSource.FilePath != nil {
		p.ConfigSource.FilePath = *req.ConfigSource.FilePath
	}
	if req.CheckoutSource != nil {
		if req.CheckoutSource.Provider != nil {
			p.CheckoutSource.Provider = *req.CheckoutSource.Provider
		}
		if req.CheckoutSource.Repo != nil && req.CheckoutSource.Repo.ExternalID != nil {
			p.CheckoutSource.Repo.ExternalID = *req.CheckoutSource.Repo.ExternalID
			p.CheckoutSource.Repo.FullName = fakeRepoFullName(p.CheckoutSource.Repo.ExternalID)
		}
	}

	respond(w, r, http.StatusOK, p)
}

func (s *Service) deletePipelineDefinition(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.lookupPipelineLocked(w, r)
	if !ok {
		return
	}

	delete(s.pipelines, p.ID)
	msg(w, r, http.StatusOK, "Pipeline definition deleted.")
}

// lookupPipelineLocked resolves {pipeline-definition-id} and writes the error
// response itself when it cannot. It requires s.mu to be held.
func (s *Service) lookupPipelineLocked(w http.ResponseWriter, r *http.Request) (*pipelineDefinition, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "pipeline-definition-id"))
	if err != nil {
		msg(w, r, http.StatusBadRequest, "Invalid pipeline definition ID.")
		return nil, false
	}

	p, ok := s.pipelines[id]
	if !ok || p.ProjectID != chi.URLParam(r, "project-id") {
		msg(w, r, http.StatusNotFound, "Pipeline definition not found.")
		return nil, false
	}

	return p, true
}

// pipelineHasTriggersLocked requires s.mu to be held.
func (s *Service) pipelineHasTriggersLocked(id uuid.UUID) bool {
	for _, t := range s.triggers {
		if t.PipelineID == id.String() {
			return true
		}
	}
	return false
}
