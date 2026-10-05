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

// v1ProjectSettings is the subset of PUT /api/v1.1/project/{vcs}/{org}/{repo}/settings
// this provider reads back after writing the oss feature flag.
type v1ProjectSettings struct {
	OSS          *bool `json:"oss"`
	FeatureFlags struct {
		OSS *bool `json:"oss"`
	} `json:"feature_flags"`
}

func (v v1ProjectSettings) ossValue() (bool, bool) {
	if v.OSS != nil {
		return *v.OSS, true
	}
	if v.FeatureFlags.OSS != nil {
		return *v.FeatureFlags.OSS, true
	}
	return false, false
}

// UpdateSettings - Settings are only available for standalone projects.
//
// oss is written through the v1.1 feature-flag API. The v2 settings API
// returns the field on read but rejects it on write with
// 400 Unexpected field 'advanced.oss', and that rejection fails the entire
// settings update. A repository that is not open source answers 422 from
// v1.1 and leaves the flag unchanged.
func (s *ProjectService) UpdateSettings(ctx context.Context, newSettings ProjectSettings, provider, organization, projectName string) (_ *ProjectSettings, err error) {
	requestedOSS := newSettings.Advanced.OSS
	v2Settings := newSettings
	v2Settings.Advanced.OSS = nil

	var settings ProjectSettings
	_, err = s.client.RequestHelper(ctx, http.MethodPatch, fmt.Sprintf("/project/%s/%s/%s/settings", provider, organization, projectName), v2Settings, &settings)
	if err != nil {
		return nil, err
	}

	if requestedOSS == nil {
		return &settings, nil
	}

	applied, err := s.setOSS(ctx, provider, organization, projectName, *requestedOSS, settings.Advanced.OSS)
	if err != nil {
		return nil, err
	}
	settings.Advanced.OSS = applied
	return &settings, nil
}

func (s *ProjectService) setOSS(ctx context.Context, provider, organization, projectName string, requested bool, current *bool) (*bool, error) {
	url := s.client.VersionedURL("v1.1", fmt.Sprintf("/project/%s/%s/%s/settings", provider, organization, projectName))
	body := map[string]map[string]bool{
		"feature_flags": {"oss": requested},
	}
	var resp v1ProjectSettings
	_, err := s.client.RequestHelperAbsolute(ctx, http.MethodPut, url, body, &resp)
	if err != nil {
		if ossNotSettable(err) {
			unchanged := false
			if current != nil {
				unchanged = *current
			}
			// true is left false on purpose so the resource can report that
			// CircleCI did not enable open source builds. false that did not
			// stick is a failed write.
			if !requested && requested != unchanged {
				return nil, fmt.Errorf("could not set oss to false: %w", err)
			}
			return common.Bool(unchanged), nil
		}
		return nil, err
	}

	applied, ok := resp.ossValue()
	if !ok {
		if current != nil {
			return current, nil
		}
		return common.Bool(false), nil
	}
	return &applied, nil
}

func ossNotSettable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not settable")
}
