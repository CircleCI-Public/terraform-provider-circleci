// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type project struct {
	Org            *org
	ID             uuid.UUID
	Name           string
	EnvVars        []EnvVarProject
	repoOpenSource bool
	settings       advancedSettings
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
	// RepoOpenSource reports whether the underlying repository is open source.
	// Nil defaults to true unless Name contains "closed-source", which the
	// provider tests use to exercise CircleCI leaving oss unchanged.
	RepoOpenSource *bool
	// OSS seeds the stored flag. Combining it with RepoOpenSource false gives
	// the state of a repository that was made private after open source builds
	// had been enabled, where CircleCI refuses to turn the flag back off.
	OSS bool
}

func (np NewProject) repoIsOpenSource() bool {
	if np.RepoOpenSource != nil {
		return *np.RepoOpenSource
	}

	return !strings.Contains(np.Name, "closed-source")
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

	p.repoOpenSource = np.repoIsOpenSource()
	p.settings = advancedSettings{
		ForksReceiveSecretEnvVars: true,
		OSS:                       np.OSS,
		PROnlyBranchOverrides:     []string{},
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

// projectPtrBySlugLocked requires s.mu to be held.
func (s *Service) projectPtrBySlugLocked(orgType, orgName, projectName string) (*project, error) {
	o := s.orgBySlugLocked(fmt.Sprintf("%s/%s", orgType, orgName))
	if o == nil {
		return nil, errNotFound
	}

	for _, p := range o.projects {
		if fmtProjectSlugSuffix(o.typ, p.ID, p.Name) == projectName {
			return p, nil
		}
	}

	return nil, errNotFound
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

// advancedSettings is the fake's stored project settings. Bool fields are
// non-pointers so responses always include them, matching the CircleCI API.
type advancedSettings struct {
	AutocancelBuilds           bool     `json:"autocancel_builds"`
	BuildForkPrs               bool     `json:"build_fork_prs"`
	DisableSSH                 bool     `json:"disable_ssh"`
	ForksReceiveSecretEnvVars  bool     `json:"forks_receive_secret_env_vars"`
	OSS                        bool     `json:"oss"`
	SetGithubStatus            bool     `json:"set_github_status"`
	SetupWorkflows             bool     `json:"setup_workflows"`
	WriteSettingsRequiresAdmin bool     `json:"write_settings_requires_admin"`
	PROnlyBranchOverrides      []string `json:"pr_only_branch_overrides"`
}

func (s advancedSettings) copy() advancedSettings {
	s.PROnlyBranchOverrides = slices.Clone(s.PROnlyBranchOverrides)
	return s
}

// advancedSettingsPatch uses pointers so omitted fields are left unchanged.
// oss true is applied only when the project's repository is open source.
type advancedSettingsPatch struct {
	AutocancelBuilds           *bool     `json:"autocancel_builds"`
	BuildForkPrs               *bool     `json:"build_fork_prs"`
	DisableSSH                 *bool     `json:"disable_ssh"`
	ForksReceiveSecretEnvVars  *bool     `json:"forks_receive_secret_env_vars"`
	OSS                        *bool     `json:"oss"`
	SetGithubStatus            *bool     `json:"set_github_status"`
	SetupWorkflows             *bool     `json:"setup_workflows"`
	WriteSettingsRequiresAdmin *bool     `json:"write_settings_requires_admin"`
	PROnlyBranchOverrides      *[]string `json:"pr_only_branch_overrides"`
}

type projectSettingsBody struct {
	Advanced advancedSettingsPatch `json:"advanced"`
}

type projectSettingsResponse struct {
	Advanced advancedSettings `json:"advanced"`
}

func (s *Service) readProjectSettings(orgType, orgName, projectName string) (advancedSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, err := s.projectPtrBySlugLocked(orgType, orgName, projectName)
	if err != nil {
		return advancedSettings{}, err
	}

	return p.settings.copy(), nil
}

func (s *Service) writeProjectSettings(orgType, orgName, projectName string, patch advancedSettingsPatch) (advancedSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.projectPtrBySlugLocked(orgType, orgName, projectName)
	if err != nil {
		return advancedSettings{}, err
	}

	if patch.AutocancelBuilds != nil {
		p.settings.AutocancelBuilds = *patch.AutocancelBuilds
	}
	if patch.BuildForkPrs != nil {
		p.settings.BuildForkPrs = *patch.BuildForkPrs
	}
	if patch.DisableSSH != nil {
		p.settings.DisableSSH = *patch.DisableSSH
	}
	if patch.ForksReceiveSecretEnvVars != nil {
		p.settings.ForksReceiveSecretEnvVars = *patch.ForksReceiveSecretEnvVars
	}
	if patch.SetGithubStatus != nil {
		p.settings.SetGithubStatus = *patch.SetGithubStatus
	}
	if patch.SetupWorkflows != nil {
		p.settings.SetupWorkflows = *patch.SetupWorkflows
	}
	if patch.WriteSettingsRequiresAdmin != nil {
		p.settings.WriteSettingsRequiresAdmin = *patch.WriteSettingsRequiresAdmin
	}
	if patch.PROnlyBranchOverrides != nil {
		p.settings.PROnlyBranchOverrides = slices.Clone(*patch.PROnlyBranchOverrides)
	}

	return p.settings.copy(), nil
}

func (s *Service) getProjectSettings(w http.ResponseWriter, r *http.Request) {
	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	settings, err := s.readProjectSettings(orgType, chi.URLParam(r, "org-name"), chi.URLParam(r, "project-name"))
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "project not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, projectSettingsResponse{Advanced: settings})
}

func (s *Service) patchProjectSettings(w http.ResponseWriter, r *http.Request) {
	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	var body projectSettingsBody
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}

	// The live v2 API returns oss on read and rejects it on write. Including
	// the field fails the whole request.
	if body.Advanced.OSS != nil {
		msg(w, r, http.StatusBadRequest, "Unexpected field 'advanced.oss'.")
		return
	}

	settings, err := s.writeProjectSettings(orgType, chi.URLParam(r, "org-name"), chi.URLParam(r, "project-name"), body.Advanced)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "project not found")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, r, http.StatusOK, projectSettingsResponse{Advanced: settings})
}

// putV1ProjectSettings writes the oss feature flag. A repository that is not
// open source is rejected with the same 422 the live API returns, and the
// stored flag is left unchanged.
func (s *Service) putV1ProjectSettings(w http.ResponseWriter, r *http.Request) {
	orgType, ok := orgTypeParam(w, r)
	if !ok {
		return
	}

	var body struct {
		FeatureFlags struct {
			OSS *bool `json:"oss"`
		} `json:"feature_flags"`
	}
	if badRequest(w, r, "bad request", render.DecodeJSON(r.Body, &body)) {
		return
	}
	if body.FeatureFlags.OSS == nil {
		msg(w, r, http.StatusBadRequest, "missing feature flag 'oss'")
		return
	}
	s.ossWrites.Add(1)

	_, err := s.setProjectOSS(orgType, chi.URLParam(r, "org-name"), chi.URLParam(r, "project-name"), *body.FeatureFlags.OSS)
	switch {
	case errors.Is(err, errNotFound):
		msg(w, r, http.StatusNotFound, "project not found")
		return
	case errors.Is(err, errOSSNotSettable):
		msg(w, r, http.StatusUnprocessableEntity, "Feature flag 'oss' is not settable for this project.")
		return
	case err != nil:
		msg(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// The live API returns 200 and a JSON empty string, not the settings object.
	respond(w, r, http.StatusOK, "")
}

// OSSWrites reports how many v1.1 feature-flag writes of oss the fake has
// accepted for decoding, rejected ones included. Writes CircleCI would refuse
// are the ones worth not sending, so the count has to include them.
func (s *Service) OSSWrites() int {
	return int(s.ossWrites.Load())
}

func (s *Service) setProjectOSS(orgType, orgName, projectName string, oss bool) (advancedSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.projectPtrBySlugLocked(orgType, orgName, projectName)
	if err != nil {
		return advancedSettings{}, err
	}
	if !p.repoOpenSource {
		return advancedSettings{}, errOSSNotSettable
	}
	p.settings.OSS = oss
	return p.settings.copy(), nil
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
