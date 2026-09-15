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

func (s *Service) setupTriggerRoutes(r chi.Router) {
	r.Post("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}/triggers", s.postTrigger)
	r.Get("/api/v2/projects/{project-id}/pipeline-definitions/{pipeline-definition-id}/triggers", s.listTriggers)
	r.Get("/api/v2/projects/{project-id}/triggers/{trigger-id}", s.getTrigger)
	r.Patch("/api/v2/projects/{project-id}/triggers/{trigger-id}", s.patchTrigger)
	r.Delete("/api/v2/projects/{project-id}/triggers/{trigger-id}", s.deleteTrigger)
}

type trigger struct {
	ID               uuid.UUID
	ProjectID        string
	PipelineID       string
	CheckoutRef      string
	ConfigRef        string
	EventName        string
	EventPreset      string
	Disabled         bool
	Provider         string
	RepoFullName     string
	RepoExternalID   string
	WebhookURL       string
	WebhookSender    string
	CronExpression   string
	AttributionActor string
	Parameters       map[string]any
	CreatedAt        time.Time
}

// triggerRequest is the body accepted by both create and update. The
// event_source members are pointers so a handler can tell an omitted field from
// an empty one, which is what the update path turns on.
type triggerRequest struct {
	CheckoutRef string         `json:"checkout_ref"`
	ConfigRef   string         `json:"config_ref"`
	EventName   string         `json:"event_name"`
	EventPreset string         `json:"event_preset"`
	Disabled    *bool          `json:"disabled"`
	Parameters  map[string]any `json:"parameters"`
	EventSource *struct {
		Provider string `json:"provider"`
		Repo     *struct {
			FullName   string `json:"full_name"`
			ExternalID string `json:"external_id"`
		} `json:"repo"`
		Webhook *struct {
			URL    string `json:"url"`
			Sender string `json:"sender"`
		} `json:"webhook"`
		Schedule *struct {
			CronExpression   string `json:"cron_expression"`
			AttributionActor string `json:"attribution_actor"`
		} `json:"schedule"`
	} `json:"event_source"`
}

// triggerResponse mirrors the real API, which always includes checkout_ref and
// config_ref and reports them as "" when they were never set.
type triggerResponse struct {
	ID          uuid.UUID              `json:"id"`
	CreatedAt   time.Time              `json:"created_at"`
	Name        string                 `json:"name"`
	EventName   string                 `json:"event_name"`
	CheckoutRef string                 `json:"checkout_ref"`
	ConfigRef   string                 `json:"config_ref"`
	EventPreset string                 `json:"event_preset,omitempty"`
	Disabled    bool                   `json:"disabled"`
	Parameters  map[string]any         `json:"parameters,omitempty"`
	EventSource triggerEventSourceResp `json:"event_source"`
}

type triggerEventSourceResp struct {
	Provider string `json:"provider,omitempty"`
	Repo     *struct {
		FullName   string `json:"full_name,omitempty"`
		ExternalID string `json:"external_id,omitempty"`
	} `json:"repo,omitempty"`
	Webhook *struct {
		URL    string `json:"url,omitempty"`
		Sender string `json:"sender,omitempty"`
	} `json:"webhook,omitempty"`
	Schedule *struct {
		CronExpression   string `json:"cron_expression,omitempty"`
		AttributionActor struct {
			ID string `json:"id,omitempty"`
		} `json:"attribution_actor"`
	} `json:"schedule,omitempty"`
}

func (t *trigger) response() triggerResponse {
	resp := triggerResponse{
		ID:          t.ID,
		CreatedAt:   t.CreatedAt,
		EventName:   t.EventName,
		CheckoutRef: t.CheckoutRef,
		ConfigRef:   t.ConfigRef,
		EventPreset: t.EventPreset,
		Disabled:    t.Disabled,
		Parameters:  t.Parameters,
	}
	resp.EventSource.Provider = t.Provider

	switch t.Provider {
	case "github_app", "github_server":
		resp.EventSource.Repo = &struct {
			FullName   string `json:"full_name,omitempty"`
			ExternalID string `json:"external_id,omitempty"`
		}{FullName: t.RepoFullName, ExternalID: t.RepoExternalID}
	case "webhook":
		resp.EventSource.Webhook = &struct {
			URL    string `json:"url,omitempty"`
			Sender string `json:"sender,omitempty"`
		}{URL: t.WebhookURL, Sender: t.WebhookSender}
	case "schedule":
		resp.EventSource.Schedule = &struct {
			CronExpression   string `json:"cron_expression,omitempty"`
			AttributionActor struct {
				ID string `json:"id,omitempty"`
			} `json:"attribution_actor"`
		}{CronExpression: t.CronExpression}
		resp.EventSource.Schedule.AttributionActor.ID = t.AttributionActor
	}

	return resp
}

// Handlers below here

func (s *Service) postTrigger(w http.ResponseWriter, r *http.Request) {
	var req triggerRequest
	if badRequest(w, r, "Invalid request body.", render.DecodeJSON(r.Body, &req)) {
		return
	}

	if req.EventSource == nil || req.EventSource.Provider == "" {
		msg(w, r, http.StatusBadRequest, "Missing field 'event_source.provider'")
		return
	}

	t := &trigger{
		ID:          uuid.New(),
		ProjectID:   chi.URLParam(r, "project-id"),
		PipelineID:  chi.URLParam(r, "pipeline-definition-id"),
		CheckoutRef: req.CheckoutRef,
		ConfigRef:   req.ConfigRef,
		EventName:   req.EventName,
		EventPreset: req.EventPreset,
		Provider:    req.EventSource.Provider,
		Parameters:  req.Parameters,
		CreatedAt:   time.Now(),
	}
	if req.Disabled != nil {
		t.Disabled = *req.Disabled
	}

	switch {
	case req.EventSource.Repo != nil:
		// The real API requires event_source.repo on create for GitHub
		// providers and resolves the full name from the external ID.
		t.RepoExternalID = req.EventSource.Repo.ExternalID
		t.RepoFullName = fmt.Sprintf("fake-org/repo-%s", t.RepoExternalID)
	case req.EventSource.Webhook != nil:
		t.WebhookSender = req.EventSource.Webhook.Sender
		t.WebhookURL = fmt.Sprintf("https://fake.circleci.test/private/webhook/%s", uuid.New())
	case req.EventSource.Schedule != nil:
		t.CronExpression = req.EventSource.Schedule.CronExpression
		t.AttributionActor = req.EventSource.Schedule.AttributionActor
	}

	s.mu.Lock()
	s.triggers[t.ID] = t
	s.mu.Unlock()

	respond(w, r, http.StatusOK, t.response())
}

func (s *Service) listTriggers(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "pipeline-definition-id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]triggerResponse, 0, len(s.triggers))
	for _, t := range s.triggers {
		if t.PipelineID == pipelineID {
			items = append(items, t.response())
		}
	}

	respond(w, r, http.StatusOK, newListResponse(items))
}

func (s *Service) getTrigger(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.lookupTriggerLocked(w, r)
	if !ok {
		return
	}

	respond(w, r, http.StatusOK, t.response())
}

// patchTrigger rejects the fields the real update endpoint declares create-only.
// Sending event_source.repo is what produced the original
// "400 Unexpected field 'event_source.repo'".
func (s *Service) patchTrigger(w http.ResponseWriter, r *http.Request) {
	var req triggerRequest
	if badRequest(w, r, "Invalid request body.", render.DecodeJSON(r.Body, &req)) {
		return
	}

	if req.EventSource != nil {
		switch {
		case req.EventSource.Repo != nil:
			msg(w, r, http.StatusBadRequest, "Unexpected field 'event_source.repo'")
			return
		case req.EventSource.Provider != "":
			msg(w, r, http.StatusBadRequest, "Unexpected field 'event_source.provider'")
			return
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.lookupTriggerLocked(w, r)
	if !ok {
		return
	}

	if req.CheckoutRef != "" {
		t.CheckoutRef = req.CheckoutRef
	}
	if req.ConfigRef != "" {
		t.ConfigRef = req.ConfigRef
	}
	if req.EventName != "" {
		t.EventName = req.EventName
	}
	if req.EventPreset != "" {
		t.EventPreset = req.EventPreset
	}
	if req.Disabled != nil {
		t.Disabled = *req.Disabled
	}
	if req.Parameters != nil {
		t.Parameters = req.Parameters
	}
	if req.EventSource != nil {
		if req.EventSource.Webhook != nil {
			t.WebhookSender = req.EventSource.Webhook.Sender
		}
		if req.EventSource.Schedule != nil {
			t.CronExpression = req.EventSource.Schedule.CronExpression
			t.AttributionActor = req.EventSource.Schedule.AttributionActor
		}
	}

	respond(w, r, http.StatusOK, t.response())
}

func (s *Service) deleteTrigger(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.lookupTriggerLocked(w, r)
	if !ok {
		return
	}

	delete(s.triggers, t.ID)
	msg(w, r, http.StatusOK, "Trigger deleted.")
}

// lookupTriggerLocked resolves {trigger-id} and writes the error response
// itself when it cannot. It requires s.mu to be held.
func (s *Service) lookupTriggerLocked(w http.ResponseWriter, r *http.Request) (*trigger, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "trigger-id"))
	if err != nil {
		msg(w, r, http.StatusBadRequest, "Invalid trigger ID.")
		return nil, false
	}

	t, ok := s.triggers[id]
	if !ok {
		msg(w, r, http.StatusNotFound, "Trigger not found.")
		return nil, false
	}

	return t, true
}
