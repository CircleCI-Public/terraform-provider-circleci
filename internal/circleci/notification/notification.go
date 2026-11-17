// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

// Package notification manages a project's Slack notifications through the
// V3 API: the Slack channel CircleCI posts to, and which kinds of
// notification it posts.
package notification

import (
	"context"
	"net/http"
	"net/url"
	"slices"

	"terraform-provider-circleci/internal/circleci/client"
)

const (
	channelTypeSlack = "slack"
	scopeProject     = "project"
	channelsPath     = "/notification/channel-configs"
	preferencesPath  = "/notification/preferences"
)

// ChannelConfig is the Slack channel a project's notifications go to.
type ChannelConfig struct {
	ID          string
	OrgID       string
	ProjectID   string
	ChannelType string
	// Target is the Slack channel ID.
	Target  string
	Enabled bool
	// ChannelName and DisableReason are nil when CircleCI does not report
	// them.
	ChannelName   *string
	DisableReason *string
}

// Preference is one kind of notification and whether a project has it on.
type Preference struct {
	ID               string
	Name             string
	Channel          string
	GroupName        string
	SectionName      string
	UserConfigurable bool
	Enabled          bool
}

type Service struct {
	client *client.Client
}

func NewService(c *client.Client) *Service {
	return &Service{client: c}
}

type projectReferences struct {
	Org     client.V3Ref `json:"org"`
	Project client.V3Ref `json:"project"`
}

type channelAttributes struct {
	ChannelType   string  `json:"channel_type"`
	Target        string  `json:"target"`
	IsEnabled     bool    `json:"is_enabled"`
	ChannelName   *string `json:"channel_name"`
	DisableReason *string `json:"disable_reason"`
}

type channelEntity = client.V3Entity[channelAttributes, projectReferences]

func fromChannelEntity(e channelEntity) *ChannelConfig {
	return &ChannelConfig{
		ID:            e.ID,
		OrgID:         e.References.Org.ID,
		ProjectID:     e.References.Project.ID,
		ChannelType:   e.Attributes.ChannelType,
		Target:        e.Attributes.Target,
		Enabled:       e.Attributes.IsEnabled,
		ChannelName:   e.Attributes.ChannelName,
		DisableReason: e.Attributes.DisableReason,
	}
}

func channelPath(id string) string {
	return channelsPath + "/" + url.PathEscape(id)
}

func projectFilter(orgID, projectID string) url.Values {
	return url.Values{
		"filter[org_id]":     {orgID},
		"filter[project_id]": {projectID},
		"filter[scope]":      {scopeProject},
	}
}

// CreateSlackChannel points the project's notifications at the Slack channel
// with ID target. When enabled, CircleCI posts a welcome message into the
// channel.
func (s *Service) CreateSlackChannel(
	ctx context.Context, orgID, projectID, target string, enabled bool,
) (*ChannelConfig, error) {
	type createAttributes struct {
		ChannelType string `json:"channel_type"`
		Scope       string `json:"scope"`
		Target      string `json:"target"`
		IsEnabled   bool   `json:"is_enabled"`
	}
	req := client.V3Data[client.V3Entity[createAttributes, projectReferences]]{
		Data: client.V3Entity[createAttributes, projectReferences]{
			Attributes: createAttributes{
				ChannelType: channelTypeSlack,
				Scope:       scopeProject,
				Target:      target,
				IsEnabled:   enabled,
			},
			References: projectReferences{
				Org:     client.V3Ref{ID: orgID},
				Project: client.V3Ref{ID: projectID},
			},
		},
	}
	var resp client.V3Data[channelEntity]
	if _, err := s.client.V3(ctx, http.MethodPost, channelsPath, nil, req, &resp); err != nil {
		return nil, err
	}
	return fromChannelEntity(resp.Data), nil
}

func (s *Service) GetChannelConfig(ctx context.Context, id string) (*ChannelConfig, error) {
	var resp client.V3Data[channelEntity]
	if _, err := s.client.V3(ctx, http.MethodGet, channelPath(id), nil, nil, &resp); err != nil {
		return nil, err
	}
	return fromChannelEntity(resp.Data), nil
}

// ListProjectChannelConfigs lists the project-scoped channel configs of a
// project. The UI allows one Slack channel per project.
func (s *Service) ListProjectChannelConfigs(ctx context.Context, orgID, projectID string) ([]ChannelConfig, error) {
	items, err := client.V3ListAll[channelEntity](ctx, s.client, channelsPath, projectFilter(orgID, projectID))
	if err != nil {
		return nil, err
	}
	configs := make([]ChannelConfig, 0, len(items))
	for _, item := range items {
		configs = append(configs, *fromChannelEntity(item))
	}
	return configs, nil
}

// UpdateChannelConfig changes the channel and whether it is enabled, and reads
// the config back. Every update that leaves the config enabled makes CircleCI
// post its welcome message again, even when nothing changed, so callers
// should only update when something did.
func (s *Service) UpdateChannelConfig(ctx context.Context, id, target string, enabled bool) (*ChannelConfig, error) {
	// The update endpoint only accepts target and is_enabled, and takes them
	// at the top level rather than wrapped in data. That is how the working
	// restapi configuration sends it; it was observed this way, not
	// documented.
	req := struct {
		Target    string `json:"target"`
		IsEnabled bool   `json:"is_enabled"`
	}{Target: target, IsEnabled: enabled}
	if _, err := s.client.V3(ctx, http.MethodPost, channelPath(id)+"/update", nil, req, nil); err != nil {
		return nil, err
	}
	return s.GetChannelConfig(ctx, id)
}

func (s *Service) DeleteChannelConfig(ctx context.Context, id string) error {
	_, err := s.client.V3(ctx, http.MethodDelete, channelPath(id), nil, nil, nil)
	return err
}

type preferenceAttributes struct {
	PreferenceName      string `json:"preference_name"`
	Channel             string `json:"channel"`
	GroupName           string `json:"group_name"`
	SectionName         string `json:"section_name"`
	SectionDisplayOrder int    `json:"section_display_order"`
	GroupDisplayOrder   int    `json:"group_display_order"`
	DisplayOrder        int    `json:"preference_display_order"`
	IsUserConfigurable  bool   `json:"is_user_configurable"`
	IsEnabled           bool   `json:"is_enabled"`
}

// ListProjectPreferences lists every notification preference of the project,
// in the order CircleCI displays them.
func (s *Service) ListProjectPreferences(ctx context.Context, orgID, projectID string) ([]Preference, error) {
	items, err := client.V3ListAll[client.V3Entity[preferenceAttributes, projectReferences]](
		ctx, s.client, preferencesPath, projectFilter(orgID, projectID),
	)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(items, func(a, b client.V3Entity[preferenceAttributes, projectReferences]) int {
		x, y := a.Attributes, b.Attributes
		switch {
		case x.SectionDisplayOrder != y.SectionDisplayOrder:
			return x.SectionDisplayOrder - y.SectionDisplayOrder
		case x.GroupDisplayOrder != y.GroupDisplayOrder:
			return x.GroupDisplayOrder - y.GroupDisplayOrder
		default:
			return x.DisplayOrder - y.DisplayOrder
		}
	})
	prefs := make([]Preference, 0, len(items))
	for _, item := range items {
		prefs = append(prefs, Preference{
			ID:               item.ID,
			Name:             item.Attributes.PreferenceName,
			Channel:          item.Attributes.Channel,
			GroupName:        item.Attributes.GroupName,
			SectionName:      item.Attributes.SectionName,
			UserConfigurable: item.Attributes.IsUserConfigurable,
			Enabled:          item.Attributes.IsEnabled,
		})
	}
	return prefs, nil
}

// UpdateProjectPreferences turns each preference in updates on or off in one
// bulk request. Preferences left out of updates are unchanged.
func (s *Service) UpdateProjectPreferences(ctx context.Context, orgID, projectID string, updates map[string]bool) error {
	if len(updates) == 0 {
		return nil
	}
	type update struct {
		PreferenceID string `json:"preference_id"`
		IsEnabled    bool   `json:"is_enabled"`
	}
	type attributes struct {
		Scope   string   `json:"scope"`
		Updates []update `json:"updates"`
	}
	ids := make([]string, 0, len(updates))
	for id := range updates {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	attrs := attributes{Scope: scopeProject, Updates: make([]update, 0, len(ids))}
	for _, id := range ids {
		attrs.Updates = append(attrs.Updates, update{PreferenceID: id, IsEnabled: updates[id]})
	}
	req := client.V3Data[client.V3Entity[attributes, projectReferences]]{
		Data: client.V3Entity[attributes, projectReferences]{
			Attributes: attrs,
			References: projectReferences{
				Org:     client.V3Ref{ID: orgID},
				Project: client.V3Ref{ID: projectID},
			},
		},
	}
	_, err := s.client.V3(ctx, http.MethodPost, preferencesPath, nil, req, nil)
	return err
}
