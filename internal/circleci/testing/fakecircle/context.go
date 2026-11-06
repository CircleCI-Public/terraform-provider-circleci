// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type context struct {
	ID        uuid.UUID
	Name      string
	Org       *org
	EnvVars   []EnvVarContext
	CreatedAt time.Time

	restrictions []contextRestriction
}

type contextRestriction struct {
	ID    uuid.UUID
	Type  string
	Value string
}

func (c *context) addEnv(ev NewEnvVarContext) (EnvVarContext, error) {
	if slices.ContainsFunc(c.EnvVars, func(e EnvVarContext) bool {
		return e.Variable == ev.Variable
	}) {
		return EnvVarContext{}, errDuplicate
	}

	now := time.Now()
	e := EnvVarContext{
		Variable:  ev.Variable,
		Value:     ev.Value,
		UpdatedAt: now,
		CreatedAt: now,
	}
	c.EnvVars = append(c.EnvVars, e)
	return e, nil
}

func (c *context) deleteEnv(ev string) {
	c.EnvVars = slices.DeleteFunc(c.EnvVars, func(e EnvVarContext) bool {
		return e.Variable == ev
	})
}

type NewContext struct {
	OrgID uuid.UUID
	Name  string
}

type Context struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

func (s *Service) AddContext(c NewContext) (Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orgs[c.OrgID]
	if !ok {
		return Context{}, errNotFound
	}

	orgCtx := &context{
		Name:      c.Name,
		Org:       o,
		ID:        uuid.New(),
		CreatedAt: time.Now(),
	}

	o.contexts[orgCtx.ID] = orgCtx
	s.contexts[orgCtx.ID] = orgCtx
	return Context{
		ID:        orgCtx.ID,
		Name:      orgCtx.Name,
		CreatedAt: orgCtx.CreatedAt,
	}, nil
}

func (s *Service) getContext(id uuid.UUID) (context, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	envCtx, ok := s.contexts[id]
	if !ok {
		return context{}, false
	}

	return context{
		ID:        envCtx.ID,
		Name:      envCtx.Name,
		EnvVars:   slices.Clone(envCtx.EnvVars),
		CreatedAt: envCtx.CreatedAt,
	}, true
}

func (s *Service) deleteEnvContext(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deleteEnvContextLocked(id)
}

// deleteEnvContextLocked requires s.mu to be held for writing.
func (s *Service) deleteEnvContextLocked(id uuid.UUID) error {
	foundCtx, ok := s.contexts[id]
	if !ok {
		return errNotFound
	}

	_, ok = foundCtx.Org.contexts[id]
	if !ok {
		return errNotFound
	}

	delete(foundCtx.Org.contexts, id)
	delete(s.contexts, id)

	return nil
}

type NewEnvVarContext struct {
	Variable string
	Value    string
}

type EnvVarContext struct {
	Variable  string
	Value     string
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (s *Service) AddContextEnv(contextID uuid.UUID, ev NewEnvVarContext) (EnvVarContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	envCtx, ok := s.contexts[contextID]
	if !ok {
		return EnvVarContext{}, errNotFound
	}

	return envCtx.addEnv(ev)
}

func (s *Service) deleteContextEnvVar(id uuid.UUID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	envCtx, ok := s.contexts[id]
	if !ok {
		return errNotFound
	}

	envCtx.deleteEnv(name)
	return nil
}

// Handlers below here

func (s *Service) postContext(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	type owner struct {
		ID   uuid.UUID `json:"id"`
		Type string    `json:"type"`
	}
	var body struct {
		Name  string `json:"name"`
		Owner owner  `json:"owner"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	envCtx, err := s.AddContext(NewContext{
		OrgID: body.Owner.ID,
		Name:  body.Name,
	})
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusBadRequest, "context not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, response(envCtx))
}

func (s *Service) deleteContext(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad context ID", err) {
		return
	}

	err = s.deleteEnvContext(id)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusBadRequest, "context not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	msg(w, r, http.StatusOK, "ok")
}

func (s *Service) getContextByID(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	id, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad context ID", err) {
		return
	}

	envCtx, ok := s.getContext(id)
	if !ok {
		msg(w, r, http.StatusNotFound, "context not found")
		return
	}

	respond(w, r, http.StatusOK, response{
		ID:        envCtx.ID,
		Name:      envCtx.Name,
		CreatedAt: envCtx.CreatedAt,
	})
}

func (s *Service) getContextBySlug(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	ownerSlug := r.URL.Query().Get("owner-slug")
	if ownerSlug == "" {
		msg(w, r, http.StatusBadRequest, "missing slug")
		return
	}

	s.mu.RLock()
	o := s.orgBySlugLocked(ownerSlug)
	var res []response
	if o != nil {
		res = make([]response, 0, len(o.contexts))
		for _, orgCtx := range o.contexts {
			res = append(res, response{
				ID:        orgCtx.ID,
				Name:      orgCtx.Name,
				CreatedAt: orgCtx.CreatedAt,
			})
		}
	}
	s.mu.RUnlock()

	if o == nil {
		msg(w, r, http.StatusNotFound, "not found")
		return
	}

	respond(w, r, http.StatusOK, newListResponse(res))
}

func (s *Service) getContextEnv(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ContextID uuid.UUID `json:"context_id"`
		Variable  string    `json:"variable"`
		UpdatedAt time.Time `json:"updated_at"`
		CreatedAt time.Time `json:"created_at"`
	}

	id, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad context ID", err) {
		return
	}

	envCtx, ok := s.getContext(id)
	if !ok {
		msg(w, r, http.StatusNotFound, "context not found")
		return
	}

	res := make([]response, 0, len(envCtx.EnvVars))
	for _, ev := range envCtx.EnvVars {
		res = append(res, response{
			Variable:  ev.Variable,
			UpdatedAt: ev.UpdatedAt,
			CreatedAt: ev.CreatedAt,
			ContextID: id,
		})
	}
	respond(w, r, http.StatusOK, newListResponse(res))
}

func (s *Service) putContextEnv(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ContextID uuid.UUID `json:"context_id"`
		Variable  string    `json:"variable"`
		UpdatedAt time.Time `json:"updated_at"`
		CreatedAt time.Time `json:"created_at"`
	}

	contextID, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad request", err) {
		return
	}

	envVarName := chi.URLParam(r, "env-var")
	if envVarName == "" {
		msg(w, r, http.StatusBadRequest, "bad request")
		return
	}

	var body struct {
		Value string `json:"value"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	ev, err := s.AddContextEnv(contextID, NewEnvVarContext{
		Variable: envVarName,
		Value:    body.Value,
	})
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusBadRequest, "context not found")
		return
	case errors.Is(err, errDuplicate):
		msg(w, r, http.StatusBadRequest, "env var already exists")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, response{
		ContextID: contextID,
		Variable:  ev.Variable,
		UpdatedAt: ev.UpdatedAt,
		CreatedAt: ev.CreatedAt,
	})
}

func (s *Service) deleteContextEnv(w http.ResponseWriter, r *http.Request) {
	contextID, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad request", err) {
		return
	}

	envVarName := chi.URLParam(r, "env-var")
	if envVarName == "" {
		msg(w, r, http.StatusBadRequest, "bad request")
		return
	}

	err = s.deleteContextEnvVar(contextID, envVarName)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusBadRequest, "context not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	msg(w, r, http.StatusOK, "ok")
}

// contextRestrictionResponse is the JSON shape of a context restriction.
// project_id and name are only set for project restrictions.
type contextRestrictionResponse struct {
	ID               uuid.UUID `json:"id"`
	ContextID        uuid.UUID `json:"context_id"`
	ProjectID        string    `json:"project_id,omitempty"`
	Name             string    `json:"name,omitempty"`
	RestrictionType  string    `json:"restriction_type"`
	RestrictionValue string    `json:"restriction_value"`
}

// restrictionResponseLocked requires s.mu to be held.
func (s *Service) restrictionResponseLocked(c *context, cr contextRestriction) contextRestrictionResponse {
	res := contextRestrictionResponse{
		ID:               cr.ID,
		ContextID:        c.ID,
		RestrictionType:  cr.Type,
		RestrictionValue: cr.Value,
	}
	if cr.Type == "project" {
		if id, err := uuid.Parse(cr.Value); err == nil {
			if p, ok := s.projects[id]; ok {
				res.ProjectID = cr.Value
				res.Name = p.Name
			}
		}
	}
	return res
}

func (s *Service) postContextRestriction(w http.ResponseWriter, r *http.Request) {
	contextID, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad context ID", err) {
		return
	}

	var body struct {
		RestrictionType  string `json:"restriction_type"`
		RestrictionValue string `json:"restriction_value"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.contexts[contextID]
	if !ok {
		msg(w, r, http.StatusNotFound, "context not found")
		return
	}

	cr := contextRestriction{ID: uuid.New(), Type: body.RestrictionType, Value: body.RestrictionValue}
	c.restrictions = append(c.restrictions, cr)
	respond(w, r, http.StatusOK, s.restrictionResponseLocked(c, cr))
}

func (s *Service) getContextRestrictions(w http.ResponseWriter, r *http.Request) {
	contextID, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad context ID", err) {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.contexts[contextID]
	if !ok {
		msg(w, r, http.StatusNotFound, "context not found")
		return
	}

	res := make([]contextRestrictionResponse, 0, len(c.restrictions))
	for _, cr := range c.restrictions {
		res = append(res, s.restrictionResponseLocked(c, cr))
	}
	respond(w, r, http.StatusOK, newListResponse(res))
}

func (s *Service) deleteContextRestriction(w http.ResponseWriter, r *http.Request) {
	contextID, err := uuid.Parse(chi.URLParam(r, "context-id"))
	if badRequest(w, r, "bad context ID", err) {
		return
	}
	restrictionID, err := uuid.Parse(chi.URLParam(r, "restriction-id"))
	if badRequest(w, r, "bad restriction ID", err) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.contexts[contextID]
	if !ok {
		msg(w, r, http.StatusNotFound, "context not found")
		return
	}

	before := len(c.restrictions)
	c.restrictions = slices.DeleteFunc(c.restrictions, func(cr contextRestriction) bool {
		return cr.ID == restrictionID
	})
	if len(c.restrictions) == before {
		msg(w, r, http.StatusNotFound, "restriction not found")
		return
	}

	msg(w, r, http.StatusOK, "ok")
}
