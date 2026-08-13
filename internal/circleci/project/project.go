// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package project

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
)

type Project struct {
	Id               string         `json:"id"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	OrganizationName string         `json:"organization_name"`
	OrganizationSlug string         `json:"organization_slug"`
	OrganizationId   string         `json:"organization_id"`
	VcsInfo          common.VcsInfo `json:"vcs_info"`
}

type AdvanceSettings struct {
	AutocancelBuilds           *bool    `json:"autocancel_builds,omitempty"`
	BuildForkPrs               *bool    `json:"build_fork_prs,omitempty"`
	DisableSSH                 *bool    `json:"disable_ssh,omitempty"`
	ForksReceiveSecretEnvVars  *bool    `json:"forks_receive_secret_env_vars,omitempty"`
	OSS                        *bool    `json:"oss,omitempty"`
	SetGithubStatus            *bool    `json:"set_github_status,omitempty"`
	SetupWorkflows             *bool    `json:"setup_workflows,omitempty"`
	WriteSettingsRequiresAdmin *bool    `json:"write_settings_requires_admin,omitempty"`
	PROnlyBranchOverrides      []string `json:"pr_only_branch_overrides,omitempty"`
}

type ProjectSettings struct {
	Advanced AdvanceSettings `json:"advanced"`
}

type ProjectService struct {
	client *client.Client
}

func NewProjectService(c *client.Client) *ProjectService {
	return &ProjectService{client: c}
}

func (s *ProjectService) Get(ctx context.Context, slug string) (_ *Project, err error) {
	var project Project
	_, err = s.client.RequestHelper(ctx, http.MethodGet, "/project/"+slug, nil, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (s *ProjectService) Create(ctx context.Context, projectName, organizationID string) (_ *Project, err error) {
	payload := map[string]string{
		"name": projectName,
	}
	var project Project
	_, err = s.client.RequestHelper(ctx, http.MethodPost, fmt.Sprintf("/organization/%s/project", organizationID), payload, &project)
	if err != nil {
		return nil, err
	}

	slug := strings.Split(project.Slug, "/")
	if len(slug) == 3 && slug[1] == project.OrganizationName {
		orgName := slug[1]
		// TODO: The URL here probably needs to be derived from the configured host for on-premise support
		url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/follow", strings.ToLower(project.VcsInfo.Provider), orgName, project.Name)
		_, err = s.client.RequestHelperAbsolute(ctx, http.MethodPost, url, nil, nil)
		if err != nil {
			return nil, err
		}
	}
	return &project, nil
}

// Delete - Only standalone projects can be deleted.
func (s *ProjectService) Delete(ctx context.Context, slug string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, fmt.Sprintf("/project/%s", slug), nil, nil)
	return err
}

// GetSettings - Settings are only available for standalone projects.
func (s *ProjectService) GetSettings(ctx context.Context, provider, organization, project string) (_ *ProjectSettings, err error) {
	var settings ProjectSettings
	_, err = s.client.RequestHelper(ctx, http.MethodGet, fmt.Sprintf("/project/%s/%s/%s/settings", provider, organization, project), nil, &settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateSettings - Settings are only available for standalone projects.
func (s *ProjectService) UpdateSettings(ctx context.Context, newSettings ProjectSettings, provider, organization, project string) (_ *ProjectSettings, err error) {
	var settings ProjectSettings
	_, err = s.client.RequestHelper(ctx, http.MethodPatch, fmt.Sprintf("/project/%s/%s/%s/settings", provider, organization, project), newSettings, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}
