// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"cmp"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// contextPageSize is deliberately small so the clients' pagination is
// exercised by any test that lists more than a couple of contexts.
const contextPageSize = 2

type context struct {
	ID           uuid.UUID
	Name         string
	Org          *org
	EnvVars      []EnvVarContext
	Restrictions []ContextRestriction
	CreatedAt    time.Time
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
		ID:           envCtx.ID,
		Name:         envCtx.Name,
		EnvVars:      slices.Clone(envCtx.EnvVars),
		Restrictions: slices.Clone(envCtx.Restrictions),
		CreatedAt:    envCtx.CreatedAt,
	}, true
}

type NewContextRestriction struct {
	Name  string
	Type  string
	Value string
}

type ContextRestriction struct {
	ID    uuid.UUID
	Name  string
	Type  string
	Value string
}

// AddContextRestriction seeds a restriction on a context.
func (s *Service) AddContextRestriction(contextID uuid.UUID, nr NewContextRestriction) (ContextRestriction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	envCtx, ok := s.contexts[contextID]
	if !ok {
		return ContextRestriction{}, errNotFound
	}

	r := ContextRestriction{
		ID:    uuid.New(),
		Name:  nr.Name,
		Type:  nr.Type,
		Value: nr.Value,
	}
	envCtx.Restrictions = append(envCtx.Restrictions, r)
	return r, nil
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

// listContexts lists an owner's contexts, identified by either the owner-id or
// the owner-slug query parameter, a page of contextPageSize at a time.
func (s *Service) listContexts(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	query := r.URL.Query()
	ownerID := query.Get("owner-id")
	ownerSlug := query.Get("owner-slug")
	if (ownerID == "") == (ownerSlug == "") {
		msg(w, r, http.StatusBadRequest, "specify exactly one of owner-id or owner-slug")
		return
	}

	offset := 0
	if tok := query.Get("page-token"); tok != "" {
		var err error
		offset, err = strconv.Atoi(tok)
		if badRequest(w, r, "bad page token", err) {
			return
		}
	}

	s.mu.RLock()
	var o *org
	if ownerID != "" {
		id, err := uuid.Parse(ownerID)
		if err != nil {
			s.mu.RUnlock()
			msg(w, r, http.StatusBadRequest, "bad owner ID")
			return
		}
		o = s.orgs[id]
	} else {
		o = s.orgBySlugLocked(ownerSlug)
	}
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

	slices.SortFunc(res, func(a, b response) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID.String(), b.ID.String()))
	})
	respond(w, r, http.StatusOK, newPage(res, offset, contextPageSize))
}

func (s *Service) getContextRestrictions(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID               uuid.UUID `json:"id"`
		ContextID        uuid.UUID `json:"context_id"`
		Name             string    `json:"name"`
		RestrictionType  string    `json:"restriction_type"`
		RestrictionValue string    `json:"restriction_value"`
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

	res := make([]response, 0, len(envCtx.Restrictions))
	for _, rs := range envCtx.Restrictions {
		res = append(res, response{
			ID:               rs.ID,
			ContextID:        id,
			Name:             rs.Name,
			RestrictionType:  rs.Type,
			RestrictionValue: rs.Value,
		})
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
