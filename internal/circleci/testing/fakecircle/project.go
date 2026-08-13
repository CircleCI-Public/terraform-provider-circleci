// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type project struct {
	Org     *org
	ID      uuid.UUID
	Name    string
	EnvVars []EnvVarProject
}

func (p *project) ToProject() Project {
	o := p.Org
	orgSlug := fmtOrgSlug(o.typ, o.id, o.name)
	p2 := Project{
		ID:   p.ID,
		Name: p.Name,
		Slug: orgSlug + "/" + fmtProjectSlugSuffix(o.typ, p.ID, p.Name),
		Org: Org{
			ID:   o.id,
			Type: o.typ,
			Name: o.name,
			Slug: orgSlug,
		},
	}
	return p2
}

type NewProject struct {
	OrgID uuid.UUID
	Name  string
}

type Project struct {
	ID   uuid.UUID
	Name string
	Slug string

	Org Org
}

func (s *Service) AddProject(np NewProject) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orgs[np.OrgID]
	if !ok {
		return Project{}, errNotFound
	}

	p, err := o.addProject(np)
	if err != nil {
		return Project{}, err
	}

	s.projects[p.ID] = p
	return p.ToProject(), nil
}

func (s *Service) Project(id uuid.UUID) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.projects[id]
	if !ok {
		return Project{}, errNotFound
	}

	return p.ToProject(), nil
}

// projectBySlugLocked requires s.mu to be held.
func (s *Service) projectBySlugLocked(orgType, orgName, projectName string) (Project, error) {
	o := s.orgBySlugLocked(fmt.Sprintf("%s/%s", orgType, orgName))
	if o == nil {
		return Project{}, errNotFound
	}

	for _, p := range o.projects {
		if fmtProjectSlugSuffix(o.typ, p.ID, p.Name) == projectName {
			return p.ToProject(), nil
		}
	}

	return Project{}, errNotFound
}

func (s *Service) projectBySlug(orgType, orgName, projectName string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.projectBySlugLocked(orgType, orgName, projectName)
}

func (s *Service) deleteProjectBySlug(orgType, orgName, projectName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.projectBySlugLocked(orgType, orgName, projectName)
	if err != nil {
		return nil
	}

	o := s.orgs[p.Org.ID]
	o.deleteProject(p.ID)
	delete(s.projects, p.ID)
	return nil
}

// handlers below here

func (s *Service) postProject(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
		Slug string    `json:"slug"`

		OrganizationName string    `json:"organization_name"`
		OrganizationSlug string    `json:"organization_slug"`
		OrganizationID   uuid.UUID `json:"organization_id"`

		VcsInfo VcsInfo `json:"vcs_info"`
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "org-id"))
	if badRequest(w, r, "bad org ID", err) {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	prj, err := s.AddProject(NewProject{
		OrgID: orgID,
		Name:  body.Name,
	})
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "org not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, response{
		ID:   prj.ID,
		Name: prj.Name,
		Slug: prj.Slug,

		OrganizationName: prj.Org.Name,
		OrganizationSlug: prj.Org.Slug,
		OrganizationID:   prj.Org.ID,

		VcsInfo: VcsInfo{
			VcsURL:        "git://github.com/dummy-value",
			Provider:      prj.Org.Type,
			DefaultBranch: "main",
		},
	})
}

type VcsInfo struct {
	VcsURL        string `json:"vcs_url"`
	Provider      string `json:"provider"`
	DefaultBranch string `json:"default_branch"`
}

// orgTypeParam reads the org-type path segment, rejecting unknown providers.
func orgTypeParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	orgType := chi.URLParam(r, "org-type")
	switch orgType {
	case TypeGitHub, TypeBitbucket, TypeCircleCI:
		return orgType, true
	default:
		msg(w, r, http.StatusBadRequest, "invalid org type")
		return "", false
	}
}

func (s *Service) getProject(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
		Slug string    `json:"slug"`

		OrganizationName string    `json:"organization_name"`
		OrganizationSlug string    `json:"organization_slug"`
		OrganizationID   uuid.UUID `json:"organization_id"`

		VcsInfo VcsInfo `json:"vcs_info"`
	}

	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	orgName := chi.URLParam(r, "org-name")
	projectName := chi.URLParam(r, "project-name")
	prj, err := s.projectBySlug(orgType, orgName, projectName)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "project not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, response{
		ID:   prj.ID,
		Name: prj.Name,
		Slug: prj.Slug,

		OrganizationName: prj.Org.Name,
		OrganizationSlug: prj.Org.Slug,
		OrganizationID:   prj.Org.ID,

		VcsInfo: VcsInfo{
			VcsURL:        "git://github.com/dummy-value",
			Provider:      prj.Org.Type,
			DefaultBranch: "main",
		},
	})
}

func (s *Service) deleteProject(w http.ResponseWriter, r *http.Request) {
	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	orgName := chi.URLParam(r, "org-name")
	projectName := chi.URLParam(r, "project-name")
	err := s.deleteProjectBySlug(orgType, orgName, projectName)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "project not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	msg(w, r, http.StatusOK, "ok")
}

type NewEnvVarProject struct {
	Name  string
	Value string
}

type EnvVarProject struct {
	Name      string
	Value     string
	CreatedAt time.Time
}

func (p *project) addEnv(ev NewEnvVarProject) (EnvVarProject, error) {
	if slices.ContainsFunc(p.EnvVars, func(e EnvVarProject) bool {
		return e.Name == ev.Name
	}) {
		return EnvVarProject{}, errDuplicate
	}

	now := time.Now()
	e := EnvVarProject{
		Name:      ev.Name,
		Value:     ev.Value,
		CreatedAt: now,
	}
	p.EnvVars = append(p.EnvVars, e)
	return e, nil
}

func (p *project) deleteEnv(ev string) {
	p.EnvVars = slices.DeleteFunc(p.EnvVars, func(e EnvVarProject) bool {
		return e.Name == ev
	})
}

func (s *Service) AddProjectEnv(projectID uuid.UUID, ev NewEnvVarProject) (EnvVarProject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	envPrj, ok := s.projects[projectID]
	if !ok {
		return EnvVarProject{}, errNotFound
	}

	return envPrj.addEnv(ev)
}

// projectEnv returns a copy of a project's environment variables.
func (s *Service) projectEnv(id uuid.UUID) ([]EnvVarProject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	envPrj, ok := s.projects[id]
	if !ok {
		return nil, false
	}

	return slices.Clone(envPrj.EnvVars), true
}

func (s *Service) deleteProjectEnvVar(id uuid.UUID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	envPrj, ok := s.projects[id]
	if !ok {
		return errNotFound
	}

	envPrj.deleteEnv(name)
	return nil
}

func (s *Service) getProjectEnv(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Value     string    `json:"value"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created-at"`
	}

	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	orgName := chi.URLParam(r, "org-name")
	projectName := chi.URLParam(r, "project-name")
	prj, err := s.projectBySlug(orgType, orgName, projectName)
	if err != nil {
		msg(w, r, http.StatusNotFound, "project not found")
		return
	}

	envVars, ok := s.projectEnv(prj.ID)
	if !ok {
		msg(w, r, http.StatusNotFound, "project not found")
		return
	}

	res := make([]response, 0, len(envVars))
	for _, ev := range envVars {
		res = append(res, response{
			Name:      ev.Name,
			CreatedAt: ev.CreatedAt,
			Value:     ev.Value,
		})
	}
	respond(w, r, http.StatusOK, newListResponse(res))
}

func (s *Service) postProjectEnv(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Name      string    `json:"name"`
		Value     string    `json:"value"`
		CreatedAt time.Time `json:"created-at"`
	}

	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	orgName := chi.URLParam(r, "org-name")
	projectName := chi.URLParam(r, "project-name")

	var body struct {
		Value string `json:"value"`
		Name  string `json:"name"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	prj, err := s.projectBySlug(orgType, orgName, projectName)
	if err != nil {
		msg(w, r, http.StatusNotFound, "project not found")
		return
	}

	ev, err := s.AddProjectEnv(prj.ID, NewEnvVarProject{
		Name:  body.Name,
		Value: body.Value,
	})
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusBadRequest, "project not found")
		return
	case errors.Is(err, errDuplicate):
		msg(w, r, http.StatusBadRequest, "env var already exists")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, response(ev))
}

func (s *Service) deleteProjectEnv(w http.ResponseWriter, r *http.Request) {
	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	orgName := chi.URLParam(r, "org-name")
	projectName := chi.URLParam(r, "project-name")
	prj, err := s.projectBySlug(orgType, orgName, projectName)
	if err != nil {
		msg(w, r, http.StatusNotFound, "project not found")
		return
	}

	envVarName := chi.URLParam(r, "env-var")
	if envVarName == "" {
		msg(w, r, http.StatusBadRequest, "bad request")
		return
	}

	err = s.deleteProjectEnvVar(prj.ID, envVarName)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusBadRequest, "project not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	msg(w, r, http.StatusOK, "ok")
}
