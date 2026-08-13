// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package organization

import (
	"context"
	"net/http"

	"terraform-provider-circleci/internal/circleci/client"
)

type Organization struct {
	Id      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	VcsType string `json:"vcs_type,omitempty"`
	Slug    string `json:"slug,omitempty"`
}

type OrganizationService struct {
	client *client.Client
}

func NewOrganizationService(c *client.Client) *OrganizationService {
	return &OrganizationService{client: c}
}

func (s *OrganizationService) Create(ctx context.Context, name, vcsType string) (org *Organization, err error) {
	org = &Organization{}
	_, err = s.client.RequestHelper(ctx, http.MethodPost, "/organization", Organization{
		Name:    name,
		VcsType: vcsType,
	}, org)
	if err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) Get(ctx context.Context, orgID string) (*Organization, error) {
	org := &Organization{}
	_, err := s.client.RequestHelper(ctx, http.MethodGet, "/organization/"+orgID, nil, org)
	if err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) Delete(ctx context.Context, orgID string) (err error) {
	_, err = s.client.RequestHelper(ctx, http.MethodDelete, "/organization/"+orgID, nil, nil)
	return err
}
