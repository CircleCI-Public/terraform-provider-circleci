// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type resourceClass struct {
	ID            string
	ResourceClass string
	Description   string
	Tokens        []*token
}

type token struct {
	ID            string
	Nickname      string
	ResourceClass string
	Token         string
	CreatedAt     string
}

type runner struct {
	Name           string
	Hostname       string
	IP             string
	Version        string
	Status         string
	ResourceClass  string
	FirstConnected string
	LastConnected  string
	LastUsed       string
}

func (s *Service) setupRunnerRoutes(r chi.Router) {
	r.Get("/api/v3/runner", s.listRunners)
	r.Get("/api/v3/runner/resource", s.listResourceClasses)
	r.Post("/api/v3/runner/resource", s.createResourceClass)
	r.Delete("/api/v3/runner/resource/{id}", s.deleteResourceClass)
	r.Delete("/api/v3/runner/resource/{id}/force", s.deleteResourceClassForce)
	r.Get("/api/v3/runner/token", s.listTokens)
	r.Post("/api/v3/runner/token", s.createToken)
	r.Delete("/api/v3/runner/token/{id}", s.deleteToken)
	r.Get("/api/v3/runner/tasks", s.getUnclaimedTaskCount)
	r.Get("/api/v3/runner/tasks/running", s.getRunningTaskCount)
}

func (s *Service) AddResourceClass(id, resourceClassIn, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.resourceClasses[id]; exists {
		return fmt.Errorf("resourceClass with id %s already exists", id)
	}
	s.resourceClasses[id] = &resourceClass{
		ID:            id,
		ResourceClass: resourceClassIn,
		Description:   description,
	}
	return nil
}

func (s *Service) listRunners(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	resourceClass := query.Get("resource-class")
	namespace := query.Get("namespace")
	orgID := query.Get("org-id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]runner, 0)
	for _, rn := range s.runners {
		if resourceClass != "" && rn.ResourceClass != resourceClass {
			continue
		}
		if namespace != "" {
			// Simple namespace matching
			continue
		}
		if orgID != "" {
			// Simple org-id matching
			continue
		}
		filtered = append(filtered, *rn)
	}

	respond(w, r, http.StatusOK, filtered)
}

func (s *Service) listResourceClasses(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	namespace := query.Get("namespace")
	orgID := query.Get("org-id")

	type responseItem struct {
		ID            string `json:"id"`
		ResourceClass string `json:"resource_class"`
		Description   string `json:"description"`
	}

	type response struct {
		Items []responseItem `json:"items"`
	}

	s.mu.RLock()
	filtered := make([]responseItem, 0)
	for _, rc := range s.resourceClasses {
		// Simple filtering - in a real implementation this would be more sophisticated
		if namespace != "" || orgID != "" {
			filtered = append(filtered, responseItem{
				ID:            rc.ID,
				ResourceClass: rc.ResourceClass,
				Description:   rc.Description,
			})
		}
	}
	s.mu.RUnlock()

	respond(w, r, http.StatusOK, response{
		Items: filtered,
	})
}

func (s *Service) createResourceClass(w http.ResponseWriter, r *http.Request) {
	type request struct {
		OrganizationID string `json:"org_id"`
		ResourceClass  string `json:"resource_class"`
		Description    string `json:"description"`
	}

	type response struct {
		ID            string `json:"id"`
		ResourceClass string `json:"resource_class"`
		Description   string `json:"description"`
	}

	var body request
	if err := render.DecodeJSON(r.Body, &body); err != nil {
		msg(w, r, http.StatusBadRequest, "bad request")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates
	for _, rc := range s.resourceClasses {
		if rc.ResourceClass == body.ResourceClass {
			msg(w, r, http.StatusConflict, "resource class already exists")
			return
		}
	}

	id := uuid.New().String()
	rc := &resourceClass{
		ID:            id,
		ResourceClass: body.ResourceClass,
		Description:   body.Description,
		Tokens:        make([]*token, 0),
	}
	s.resourceClasses[id] = rc

	respond(w, r, http.StatusOK, &response{
		ID:            rc.ID,
		ResourceClass: rc.ResourceClass,
		Description:   rc.Description,
	})
}

func (s *Service) deleteResourceClass(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	rc, ok := s.resourceClasses[id]
	if !ok {
		msg(w, r, http.StatusNotFound, "resource class not found")
		return
	}

	// Check if there are tokens
	if len(rc.Tokens) > 0 {
		msg(w, r, http.StatusConflict, "resource class has tokens")
		return
	}

	delete(s.resourceClasses, id)
	respond(w, r, http.StatusOK, map[string]any{"message": "deleted"})
}

func (s *Service) deleteResourceClassForce(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	rc, ok := s.resourceClasses[id]
	if !ok {
		msg(w, r, http.StatusNotFound, "resource class not found")
		return
	}

	// Delete all tokens
	for _, t := range rc.Tokens {
		delete(s.tokens, t.ID)
	}

	delete(s.resourceClasses, id)
	respond(w, r, http.StatusOK, map[string]any{"message": "deleted"})
}

func (s *Service) listTokens(w http.ResponseWriter, r *http.Request) {
	resourceClass := r.URL.Query().Get("resource-class")

	type response struct {
		ID            string `json:"id"`
		Nickname      string `json:"nickname"`
		ResourceClass string `json:"resource_class"`
		CreatedAt     string `json:"created_at"`
	}

	type responseItems struct {
		Items []response `json:"items"`
	}

	s.mu.RLock()
	filtered := make([]response, 0)
	for _, t := range s.tokens {
		if resourceClass == "" || t.ResourceClass == resourceClass {
			filtered = append(filtered, response{
				ID:            t.ID,
				Nickname:      t.Nickname,
				ResourceClass: t.ResourceClass,
				CreatedAt:     t.CreatedAt,
			})
		}
	}
	s.mu.RUnlock()

	respond(w, r, http.StatusOK, responseItems{
		Items: filtered,
	})
}

func (s *Service) createToken(w http.ResponseWriter, r *http.Request) {
	type request struct {
		OrganizationID string `json:"org_id"`
		ResourceClass  string `json:"resource_class"`
		Nickname       string `json:"nickname"`
	}

	type response struct {
		ID            string `json:"id"`
		Nickname      string `json:"nickname"`
		ResourceClass string `json:"resource_class"`
		Token         string `json:"token"`
		CreatedAt     string `json:"created_at"`
	}

	var body request
	if err := render.DecodeJSON(r.Body, &body); err != nil {
		msg(w, r, http.StatusBadRequest, "bad request")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find resource class
	var rc *resourceClass
	for _, candidate := range s.resourceClasses {
		if candidate.ResourceClass == body.ResourceClass {
			rc = candidate
			break
		}
	}

	if rc == nil {
		msg(w, r, http.StatusNotFound, "resource class not found")
		return
	}

	id := uuid.New().String()
	tokenValue := "token_" + uuid.New().String()
	t := &token{
		ID:            id,
		Nickname:      body.Nickname,
		ResourceClass: body.ResourceClass,
		Token:         tokenValue,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}
	s.tokens[id] = t
	rc.Tokens = append(rc.Tokens, t)

	respond(w, r, http.StatusOK, response{
		ID:            t.ID,
		Nickname:      t.Nickname,
		ResourceClass: t.ResourceClass,
		Token:         t.Token,
		CreatedAt:     t.CreatedAt,
	})
}

func (s *Service) deleteToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tokens[id]
	if !ok {
		msg(w, r, http.StatusNotFound, "token not found")
		return
	}

	// Remove from resource class
	for _, rc := range s.resourceClasses {
		if rc.ResourceClass == t.ResourceClass {
			rc.Tokens = slices.DeleteFunc(rc.Tokens, func(tok *token) bool {
				return tok.ID == id
			})
			break
		}
	}

	delete(s.tokens, id)
	respond(w, r, http.StatusOK, map[string]any{"message": "deleted"})
}

func (s *Service) getUnclaimedTaskCount(w http.ResponseWriter, r *http.Request) {
	type response struct {
		UnclaimedTaskCount int `json:"unclaimed_task_count"`
	}

	respond(w, r, http.StatusOK, response{
		UnclaimedTaskCount: 0,
	})
}

func (s *Service) getRunningTaskCount(w http.ResponseWriter, r *http.Request) {
	type response struct {
		RunningRunnerTasks int `json:"running_runner_tasks"`
	}

	respond(w, r, http.StatusOK, response{
		RunningRunnerTasks: 0,
	})
}
