// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

// Package orgsettings reads and changes an organization's settings through
// the V3 API, the endpoints `circleci org setting` uses.
package orgsettings

import (
	"context"
	"net/http"
	"net/url"
	"reflect"

	"terraform-provider-circleci/internal/circleci/client"
)

// Settings are the organization settings the provider manages. A nil field is
// one a request leaves alone. GET also reports is_runner_terms_of_service_accepted,
// is_running_disabled and is_code_signing_restricted_to_admins, which are
// deliberately not managed and so are not decoded here.
type Settings struct {
	EnableAIErrorSummarization        *bool `json:"enable_ai_error_summarization,omitempty"`
	EnableAIAgents                    *bool `json:"enable_ai_agents,omitempty"`
	EnableMinorAIFeatures             *bool `json:"enable_minor_ai_features,omitempty"`
	EnableUnversionedConfig           *bool `json:"enable_unversioned_config,omitempty"`
	EnableCertifiedPublicOrbs         *bool `json:"enable_certified_public_orbs,omitempty"`
	EnableUncertifiedPublicOrbs       *bool `json:"enable_uncertified_public_orbs,omitempty"`
	EnablePrivateOrbs                 *bool `json:"enable_private_orbs,omitempty"`
	EnableChunkIPRanges               *bool `json:"enable_chunk_ip_ranges,omitempty"`
	EnableMarketingTracking           *bool `json:"enable_marketing_tracking,omitempty"`
	EnablePreferencesTracking         *bool `json:"enable_preferences_tracking,omitempty"`
	EnableStatisticsTracking          *bool `json:"enable_statistics_tracking,omitempty"`
	IsPrivacyOptoutEnforced           *bool `json:"is_privacy_optout_enforced,omitempty"`
	IsUserCheckoutKeysDisabled        *bool `json:"is_user_checkout_keys_disabled,omitempty"`
	EnableImageBrownouts              *bool `json:"enable_image_brownouts,omitempty"`
	EnableResourceClassBrownouts      *bool `json:"enable_resource_class_brownouts,omitempty"`
	IsContextGroupRestrictionRequired *bool `json:"is_context_group_restriction_required,omitempty"`
}

// empty reports whether s carries no setting at all.
func (s Settings) empty() bool {
	return reflect.ValueOf(s).IsZero()
}

type Service struct {
	client *client.Client
}

func NewService(c *client.Client) *Service {
	return &Service{client: c}
}

func orgPath(orgID string) string {
	return "/orgs/" + url.PathEscape(orgID)
}

// settingsEntity is the observed GET body: attributes only, no id or
// references.
type settingsEntity = client.V3Entity[Settings, struct{}]

// Get returns the organization's current settings. The API answers 404 both
// for an organization that does not exist and, it seems, for a token that may
// not read it.
func (s *Service) Get(ctx context.Context, orgID string) (*Settings, error) {
	var resp client.V3Data[settingsEntity]
	if _, err := s.client.V3(ctx, http.MethodGet, orgPath(orgID)+"/settings", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data.Attributes, nil
}

// Update writes the non-nil settings in changes and returns the settings read
// back afterwards. With nothing to change it only reads.
func (s *Service) Update(ctx context.Context, orgID string, changes Settings) (*Settings, error) {
	if !changes.empty() {
		if err := s.write(ctx, orgID, changes); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, orgID)
}

// write sends the settings being changed as a V3 single-entity body,
// {"data":{"attributes":{...}}}. The response is ignored; callers read the
// settings back with GET.
func (s *Service) write(ctx context.Context, orgID string, changes Settings) error {
	req := client.V3Data[settingsEntity]{Data: settingsEntity{Attributes: changes}}
	_, err := s.client.V3(ctx, http.MethodPost, orgPath(orgID)+"/update-settings", nil, req, nil)
	return err
}
