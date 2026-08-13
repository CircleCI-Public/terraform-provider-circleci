// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/mr-tron/base58"
)

const (
	TypeGitHub    = "github"
	TypeBitbucket = "bitbucket"
	TypeCircleCI  = "circleci"
)

// orgBySlugLocked requires s.mu to be held.
func (s *Service) orgBySlugLocked(slug string) *org {
	for id, o := range s.orgs {
		if fmtOrgSlug(o.typ, id, o.name) == slug {
			return &org{
				id:       o.id,
				typ:      o.typ,
				name:     o.name,
				contexts: maps.Clone(o.contexts),
				projects: maps.Clone(o.projects),
			}
		}
	}
	return nil
}

type org struct {
	id       uuid.UUID
	typ      string
	name     string
	contexts map[uuid.UUID]*context
	projects map[uuid.UUID]*project
}

func (o *org) addProject(np NewProject) (*project, error) {
	for _, p := range o.projects {
		if p.Name == np.Name {
			return nil, errDuplicate
		}
	}

	p := &project{
		ID:   uuid.New(),
		Org:  o,
		Name: np.Name,
	}

	o.projects[p.ID] = p
	return p, nil
}

func (o *org) deleteProject(id uuid.UUID) {
	delete(o.projects, id)
}

type NewOrg struct {
	Type string
	Name string
}

type Org struct {
	ID   uuid.UUID
	Type string
	Name string
	Slug string
}

func fmtOrgSlug(typ string, id uuid.UUID, name string) string {
	second := name

	if typ == TypeCircleCI {
		second = base58.Encode(id[:])
	}

	return fmt.Sprintf("%s/%s", typ, second)
}

func fmtProjectSlugSuffix(typ string, id uuid.UUID, name string) string {
	if typ == TypeCircleCI {
		return base58.Encode(id[:])
	}

	return name
}

func (s *Service) AddOrg(newOrg NewOrg) (Org, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, o := range s.orgs {
		if o.name == newOrg.Name {
			return Org{}, errDuplicate
		}
	}

	o := &org{
		id:       uuid.New(),
		typ:      newOrg.Type,
		name:     newOrg.Name,
		contexts: make(map[uuid.UUID]*context),
		projects: make(map[uuid.UUID]*project),
	}
	s.orgs[o.id] = o
	return Org{
		ID:   o.id,
		Type: o.typ,
		Name: o.name,
		Slug: fmtOrgSlug(o.typ, o.id, o.name),
	}, nil
}

func (s *Service) deleteOrg(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orgs[id]
	if !ok {
		return errNotFound
	}

	for contextID := range o.contexts {
		err := s.deleteEnvContextLocked(contextID)
		if err != nil {
			return err
		}
	}

	delete(s.orgs, id)
	return nil
}

// handlers below here

func (s *Service) postOrganization(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID      uuid.UUID `json:"id"`
		Name    string    `json:"name"`
		VcsType string    `json:"vcs_type"`
		Slug    string    `json:"slug"`
	}

	var body struct {
		Type string `json:"vcs_type"`
		Name string `json:"name"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	o, err := s.AddOrg(NewOrg(body))
	switch {
	case errors.Is(err, errDuplicate):
		msg(w, r, http.StatusBadRequest, "duplicate org")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, response{
		ID:      o.ID,
		Name:    o.Name,
		VcsType: o.Type,
		Slug:    fmtOrgSlug(o.Type, o.ID, o.Name),
	})
}

func (s *Service) deleteOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "org-id"))
	if badRequest(w, r, "bad org ID", err) {
		return
	}

	err = s.deleteOrg(orgID)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "org not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	msg(w, r, http.StatusOK, "ok")
}

func (s *Service) getOrganizationByID(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID      uuid.UUID `json:"id"`
		Name    string    `json:"name"`
		VcsType string    `json:"vcs_type"`
		Slug    string    `json:"slug"`
	}

	id, err := uuid.Parse(chi.URLParam(r, "org-id"))
	if badRequest(w, r, "bad org ID", err) {
		return
	}

	s.mu.RLock()
	o, ok := s.orgs[id]
	var res response
	if ok {
		res = response{
			ID:      o.id,
			Name:    o.name,
			VcsType: o.typ,
			Slug:    fmtOrgSlug(o.typ, o.id, o.name),
		}
	}
	s.mu.RUnlock()

	if !ok {
		return
	}

	respond(w, r, http.StatusOK, res)
}
