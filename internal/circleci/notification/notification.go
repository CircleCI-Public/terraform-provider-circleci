// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

// Package notification manages a project's Slack notifications through the
// V3 API: the Slack channel CircleCI posts to, and which kinds of
// notification it posts.
package notification

import (
	"context"
	"errors"

	"terraform-provider-circleci/internal/circleci/client"
)

var errNotImplemented = errors.New("not implemented")

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

func (s *Service) CreateSlackChannel(_ context.Context, _, _, _ string, _ bool) (*ChannelConfig, error) {
	return nil, errNotImplemented
}

func (s *Service) GetChannelConfig(_ context.Context, _ string) (*ChannelConfig, error) {
	return nil, errNotImplemented
}

func (s *Service) ListProjectChannelConfigs(_ context.Context, _, _ string) ([]ChannelConfig, error) {
	return nil, errNotImplemented
}

func (s *Service) UpdateChannelConfig(_ context.Context, _, _ string, _ bool) (*ChannelConfig, error) {
	return nil, errNotImplemented
}

func (s *Service) DeleteChannelConfig(_ context.Context, _ string) error {
	return errNotImplemented
}

func (s *Service) ListProjectPreferences(_ context.Context, _, _ string) ([]Preference, error) {
	return nil, errNotImplemented
}

func (s *Service) UpdateProjectPreferences(_ context.Context, _, _ string, _ map[string]bool) error {
	return errNotImplemented
}
