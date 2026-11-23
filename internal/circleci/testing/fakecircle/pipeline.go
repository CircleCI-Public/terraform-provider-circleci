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

func (s *Service) setupPipelineV3Routes(r chi.Router) {
	r.Post("/api/v3/pipelines", s.postPipelineV3)
	r.Get("/api/v3/pipelines", s.listPipelinesV3)
	r.Get("/api/v3/pipelines/{id}", s.getPipelineV3)
	r.Post("/api/v3/pipelines/{id}/update", s.updatePipelineV3)
	r.Delete("/api/v3/pipelines/{id}", s.deletePipelineV3)
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

type vcsV3 struct {
	Provider     string `json:"provider"`
	RepoID       string `json:"repo_id"`
	RepoFullName string `json:"repo_full_name"`
}

type pipelineCreateV3 struct {
	Data struct {
		Attributes struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Config      struct {
				Type     string `json:"type"`
				FilePath string `json:"file_path"`
				FileType string `json:"file_type"`
				VCS      *vcsV3 `json:"vcs"`
			} `json:"config"`
			Checkout struct {
				VCS vcsV3 `json:"vcs"`
			} `json:"checkout"`
		} `json:"attributes"`
		References struct {
			Project v3Ref `json:"project"`
		} `json:"references"`
	} `json:"data"`
}

func (p *pipelineDef) v3() v3Entity {
	return v3Entity{
		ID: &p.ID,
		Attributes: map[string]any{
			"name":        p.Name,
			"description": p.Description,
			"created_at":  p.CreatedAt,
			"checkout": map[string]any{
				"type": "vcs",
				"vcs":  vcsV3{Provider: p.Provider, RepoID: p.CheckoutRepoID, RepoFullName: p.CheckoutRepoFullName},
			},
			"config": map[string]any{
				"type":      "vcs",
				"file_path": p.FilePath,
				"file_type": "circleci",
				"vcs":       vcsV3{Provider: p.Provider, RepoID: p.ConfigRepoID, RepoFullName: p.ConfigRepoFullName},
			},
		},
		References: map[string]any{"project": v3Ref{ID: p.ProjectID}},
	}
}

func (s *Service) postPipelineV3(w http.ResponseWriter, r *http.Request) {
	var req pipelineCreateV3
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	attrs := req.Data.Attributes
	if attrs.Name == "" {
		v3Error(w, r, http.StatusBadRequest, "name is required")
		return
	}
	if attrs.Config.Type != "vcs" || attrs.Config.VCS == nil {
		v3Error(w, r, http.StatusBadRequest, "config.vcs is required for vcs config")
		return
	}
	if attrs.Config.VCS.Provider != attrs.Checkout.VCS.Provider {
		v3Error(w, r, http.StatusBadRequest, "config.vcs.provider and checkout.vcs.provider must match")
		return
	}
	if attrs.Checkout.VCS.Provider != "forgejo" {
		v3Error(w, r, http.StatusBadRequest, "checkout.vcs.provider is not supported by this fake")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	projectID := req.Data.References.Project.ID
	if !s.forgejoConnectedLocked(projectID) {
		v3Error(w, r, http.StatusBadRequest, "the project's organization has no forgejo connection")
		return
	}
	configID, configName, ok := s.resolveForgejoRepoLocked(*attrs.Config.VCS)
	if !ok {
		v3Error(w, r, http.StatusBadRequest, "config repository not found")
		return
	}
	checkoutID, checkoutName, ok := s.resolveForgejoRepoLocked(attrs.Checkout.VCS)
	if !ok {
		v3Error(w, r, http.StatusBadRequest, "checkout repository not found")
		return
	}

	p := &pipelineDef{
		ID:                   uuid.New(),
		ProjectID:            projectID,
		Name:                 attrs.Name,
		Description:          attrs.Description,
		CreatedAt:            time.Now().UTC(),
		Provider:             "forgejo",
		FilePath:             attrs.Config.FilePath,
		ConfigRepoID:         configID,
		ConfigRepoFullName:   configName,
		CheckoutRepoID:       checkoutID,
		CheckoutRepoFullName: checkoutName,
	}
	s.pipelines[p.ID] = p
	v3Data(w, r, http.StatusCreated, p.v3())
}

func (s *Service) listPipelinesV3(w http.ResponseWriter, r *http.Request) {
	projectID, ok := filterUUID(w, r, "project_id", true)
	if !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]v3Entity, 0)
	for _, p := range s.pipelines {
		if p.ProjectID == projectID {
			items = append(items, p.v3())
		}
	}
	respond(w, r, http.StatusOK, map[string]any{"data": items, "meta": map[string]any{"total_count": len(items)}})
}

func (s *Service) getPipelineV3(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.lookupPipelineV3Locked(w, r)
	if !ok {
		return
	}
	v3Data(w, r, http.StatusOK, p.v3())
}

func (s *Service) updatePipelineV3(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Config      *struct {
			FilePath string  `json:"file_path"`
			FileType *string `json:"file_type"`
		} `json:"config"`
	}
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Name == nil && req.Description == nil && (req.Config == nil || req.Config.FilePath == "") {
		v3Error(w, r, http.StatusBadRequest,
			"at least one of name, description, config.file_path, or config.file_type must be provided")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.lookupPipelineV3Locked(w, r)
	if !ok {
		return
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Config != nil && req.Config.FilePath != "" {
		p.FilePath = req.Config.FilePath
	}
	v3Data(w, r, http.StatusOK, p.v3())
}

func (s *Service) deletePipelineV3(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.lookupPipelineV3Locked(w, r)
	if !ok {
		return
	}
	if s.pipelineHasTriggersLocked(p.ID) {
		v3Error(w, r, http.StatusBadRequest, "the pipeline cannot be deleted because it has one or more triggers associated with it")
		return
	}
	s.deletePipelineLocked(p.ID)
	noContent(w)
}

func (s *Service) lookupPipelineV3Locked(w http.ResponseWriter, r *http.Request) (*pipelineDef, bool) {
	id, ok := pathUUID(w, r)
	if !ok {
		return nil, false
	}
	projectID, ok := filterUUID(w, r, "project_id", false)
	if !ok {
		return nil, false
	}
	p, found := s.pipelines[id]
	if !found || (projectID != uuid.Nil && p.ProjectID != projectID) {
		v3Error(w, r, http.StatusNotFound, "pipeline not found")
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

// deleteForgejoPipelinesLocked removes the Forgejo pipelines, and with them
// their triggers, of every project in orgID, as deleting the org's Forgejo
// connection does.
func (s *Service) deleteForgejoPipelinesLocked(orgID uuid.UUID) {
	for id, p := range s.pipelines {
		if proj, ok := s.projects[p.ProjectID]; ok && proj.Org.id == orgID && p.Provider == "forgejo" {
			s.deletePipelineLocked(id)
		}
	}
	for id, t := range s.triggers {
		if t.Provider != "forgejo" {
			continue
		}
		projectID, err := uuid.Parse(t.ProjectID)
		if err != nil {
			continue
		}
		if proj, ok := s.projects[projectID]; ok && proj.Org.id == orgID {
			delete(s.triggers, id)
		}
	}
}

func (s *Service) forgejoConnectedLocked(projectID uuid.UUID) bool {
	proj, ok := s.projects[projectID]
	if !ok {
		return false
	}
	for _, c := range s.connections {
		if c.OrgID == proj.Org.id {
			return true
		}
	}
	return false
}

// resolveForgejoRepoLocked looks a repository up by ID or full name, as the
// real API does through the org's connection.
func (s *Service) resolveForgejoRepoLocked(v vcsV3) (string, string, bool) {
	for name, id := range s.forgejoRepos {
		if (v.RepoID != "" && v.RepoID == id) || (v.RepoID == "" && v.RepoFullName == name) {
			return id, name, true
		}
	}
	return "", "", false
}
