// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// forgejoPresets are the presets the real API can evaluate for Forgejo.
var forgejoPresets = []string{
	"all-pushes",
	"only-tags",
	"default-branch-pushes",
	"only-build-prs",
	"only-open-prs",
	"non-draft-pr-opened",
	"only-ready-for-review-prs",
	"only-build-pushes-to-non-draft-prs",
	"only-merged-prs",
	"only-merged-or-closed-prs",
}

func (s *Service) setupTriggerV3Routes(r chi.Router) {
	r.Post("/api/v3/triggers", s.postTriggerV3)
	r.Get("/api/v3/triggers", s.listTriggersV3)
	r.Get("/api/v3/triggers/{id}", s.getTriggerV3)
	r.Post("/api/v3/triggers/{id}/update", s.updateTriggerV3)
	r.Delete("/api/v3/triggers/{id}", s.deleteTriggerV3)
}

type refV3 struct {
	Ref *string `json:"ref"`
}

type triggerFilterV3 struct {
	Preset string `json:"preset,omitempty"`
}

type triggerCreateV3 struct {
	Data struct {
		Attributes struct {
			IsDisabled bool   `json:"is_disabled"`
			Checkout   *refV3 `json:"checkout"`
			Config     *refV3 `json:"config"`
			Event      struct {
				Type   string           `json:"type"`
				VCS    *vcsV3           `json:"vcs"`
				Filter *triggerFilterV3 `json:"filter"`
			} `json:"event"`
		} `json:"attributes"`
		References struct {
			Pipeline v3Ref `json:"pipeline"`
		} `json:"references"`
	} `json:"data"`
}

func (t *trigger) v3() v3Entity {
	attrs := map[string]any{
		"is_disabled": t.Disabled,
		"created_at":  t.CreatedAt,
		"event": map[string]any{
			"type":   "vcs",
			"vcs":    vcsV3{Provider: t.Provider, RepoID: t.RepoExternalID, RepoFullName: t.RepoFullName},
			"filter": triggerFilterV3{Preset: t.EventPreset},
		},
	}
	if t.CheckoutRef != "" {
		attrs["checkout"] = map[string]any{"ref": t.CheckoutRef}
	}
	if t.ConfigRef != "" {
		attrs["config"] = map[string]any{"ref": t.ConfigRef}
	}
	return v3Entity{
		ID:         &t.ID,
		Attributes: attrs,
		References: map[string]any{
			"project":  map[string]any{"id": t.ProjectID},
			"pipeline": map[string]any{"id": t.PipelineID},
		},
	}
}

func refValue(r *refV3) string {
	if r == nil || r.Ref == nil {
		return ""
	}
	return *r.Ref
}

func (s *Service) postTriggerV3(w http.ResponseWriter, r *http.Request) {
	var req triggerCreateV3
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	attrs := req.Data.Attributes
	if attrs.Event.Type != "vcs" || attrs.Event.VCS == nil {
		v3Error(w, r, http.StatusBadRequest, "event.vcs is required for vcs triggers")
		return
	}
	if attrs.Event.VCS.Provider != "forgejo" {
		v3Error(w, r, http.StatusBadRequest, "event.vcs.provider is not supported by this fake")
		return
	}
	if attrs.Event.Filter == nil || !slices.Contains(forgejoPresets, attrs.Event.Filter.Preset) {
		v3Error(w, r, http.StatusBadRequest, "unknown event key for forgejo")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pipelines[req.Data.References.Pipeline.ID]
	if !ok {
		v3Error(w, r, http.StatusNotFound, "pipeline not found")
		return
	}
	repoID, repoName, ok := s.resolveForgejoRepoLocked(*attrs.Event.VCS)
	if !ok {
		v3Error(w, r, http.StatusBadRequest, "event repository not found")
		return
	}

	t := &trigger{
		ID:             uuid.New(),
		ProjectID:      p.ProjectID.String(),
		PipelineID:     p.ID.String(),
		CheckoutRef:    refValue(attrs.Checkout),
		ConfigRef:      refValue(attrs.Config),
		EventPreset:    attrs.Event.Filter.Preset,
		Disabled:       attrs.IsDisabled,
		Provider:       "forgejo",
		RepoExternalID: repoID,
		RepoFullName:   repoName,
		CreatedAt:      time.Now().UTC(),
	}
	s.triggers[t.ID] = t
	v3Data(w, r, http.StatusCreated, t.v3())
}

func (s *Service) listTriggersV3(w http.ResponseWriter, r *http.Request) {
	projectID, ok := filterUUID(w, r, "project_id", true)
	if !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]v3Entity, 0)
	for _, t := range s.triggers {
		if t.ProjectID == projectID.String() && t.Provider == "forgejo" {
			items = append(items, t.v3())
		}
	}
	respond(w, r, http.StatusOK, map[string]any{"data": items, "meta": map[string]any{"total_count": len(items)}})
}

func (s *Service) getTriggerV3(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.lookupTriggerV3Locked(w, r)
	if !ok {
		return
	}
	v3Data(w, r, http.StatusOK, t.v3())
}

func (s *Service) updateTriggerV3(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsDisabled *bool  `json:"is_disabled"`
		Checkout   *refV3 `json:"checkout"`
		Config     *refV3 `json:"config"`
		Event      *struct {
			Filter *triggerFilterV3 `json:"filter"`
		} `json:"event"`
	}
	if err := decodeStrict(r, &req); err != nil {
		v3Error(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.lookupTriggerV3Locked(w, r)
	if !ok {
		return
	}
	if req.Event != nil && req.Event.Filter != nil {
		if !slices.Contains(forgejoPresets, req.Event.Filter.Preset) {
			v3Error(w, r, http.StatusBadRequest, "unknown event key for forgejo")
			return
		}
		t.EventPreset = req.Event.Filter.Preset
	}
	if req.IsDisabled != nil {
		t.Disabled = *req.IsDisabled
	}
	if req.Checkout != nil && req.Checkout.Ref != nil {
		t.CheckoutRef = *req.Checkout.Ref
	}
	if req.Config != nil && req.Config.Ref != nil {
		t.ConfigRef = *req.Config.Ref
	}
	v3Data(w, r, http.StatusOK, t.v3())
}

func (s *Service) deleteTriggerV3(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.lookupTriggerV3Locked(w, r)
	if !ok {
		return
	}
	delete(s.triggers, t.ID)
	noContent(w)
}

func (s *Service) lookupTriggerV3Locked(w http.ResponseWriter, r *http.Request) (*trigger, bool) {
	id, ok := pathUUID(w, r)
	if !ok {
		return nil, false
	}
	projectID, ok := filterUUID(w, r, "project_id", false)
	if !ok {
		return nil, false
	}
	t, found := s.triggers[id]
	if !found || (projectID != uuid.Nil && t.ProjectID != projectID.String()) {
		v3Error(w, r, http.StatusNotFound, "trigger not found")
		return nil, false
	}
	return t, true
}
