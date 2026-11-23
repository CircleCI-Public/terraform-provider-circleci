// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

// Package group manages an organization's RBAC groups through the V3 API:
// the groups themselves, their members, and the roles they hold on projects.
package group

import (
	"context"
	"net/http"
	"net/url"

	"terraform-provider-circleci/internal/circleci/client"
)

// ProjectRoles are the roles a group can be granted on a project.
var ProjectRoles = []string{"project-admin", "project-contributor", "project-viewer"}

type Group struct {
	ID          string
	OrgID       string
	Name        string
	Description string
}

type Member struct {
	UserID   string
	Username string
	Email    string
}

// ProjectGroup is a group that holds a role on a project.
type ProjectGroup struct {
	ID   string
	Name string
	Role string
}

type Service struct {
	client *client.Client
}

func NewService(c *client.Client) *Service {
	return &Service{client: c}
}

type groupAttributes struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type groupReferences struct {
	Org client.V3Ref `json:"org"`
}

type groupEntity = client.V3Entity[groupAttributes, groupReferences]

func fromEntity(e groupEntity) *Group {
	return &Group{
		ID:          e.ID,
		OrgID:       e.References.Org.ID,
		Name:        e.Attributes.Name,
		Description: e.Attributes.Description,
	}
}

// orgFilter is the query every group endpoint but create needs, because the
// API authorizes against the organization it names.
func orgFilter(orgID string) url.Values {
	return url.Values{"filter[org_id]": {orgID}}
}

func groupPath(id string) string {
	return "/groups/" + url.PathEscape(id)
}

func (s *Service) Create(ctx context.Context, orgID, name, description string) (*Group, error) {
	req := client.V3Data[groupEntity]{Data: groupEntity{
		Attributes: groupAttributes{Name: name, Description: description},
		References: groupReferences{Org: client.V3Ref{ID: orgID}},
	}}
	var resp client.V3Data[groupEntity]
	if _, err := s.client.V3(ctx, http.MethodPost, "/groups", nil, req, &resp); err != nil {
		return nil, err
	}
	return fromEntity(resp.Data), nil
}

func (s *Service) Get(ctx context.Context, orgID, id string) (*Group, error) {
	var resp client.V3Data[groupEntity]
	if _, err := s.client.V3(ctx, http.MethodGet, groupPath(id), orgFilter(orgID), nil, &resp); err != nil {
		return nil, err
	}
	return fromEntity(resp.Data), nil
}

func (s *Service) Update(ctx context.Context, orgID, id, name, description string) (*Group, error) {
	req := client.V3Data[client.V3Entity[groupAttributes, struct{}]]{
		Data: client.V3Entity[groupAttributes, struct{}]{
			Attributes: groupAttributes{Name: name, Description: description},
		},
	}
	var resp client.V3Data[groupEntity]
	if _, err := s.client.V3(ctx, http.MethodPut, groupPath(id), orgFilter(orgID), req, &resp); err != nil {
		return nil, err
	}
	return fromEntity(resp.Data), nil
}

func (s *Service) Delete(ctx context.Context, orgID, id string) error {
	_, err := s.client.V3(ctx, http.MethodDelete, groupPath(id), orgFilter(orgID), nil, nil)
	return err
}

func (s *Service) Members(ctx context.Context, orgID, id string) ([]Member, error) {
	type attributes struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	items, err := client.V3ListAll[client.V3Entity[attributes, struct{}]](
		ctx, s.client, groupPath(id)+"/members", orgFilter(orgID),
	)
	if err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(items))
	for _, item := range items {
		members = append(members, Member{
			UserID:   item.ID,
			Username: item.Attributes.Username,
			Email:    item.Attributes.Email,
		})
	}
	return members, nil
}

func refs(ids []string) []client.V3Ref {
	out := make([]client.V3Ref, 0, len(ids))
	for _, id := range ids {
		out = append(out, client.V3Ref{ID: id})
	}
	return out
}

func (s *Service) AddMembers(ctx context.Context, orgID, id string, userIDs ...string) error {
	req := map[string][]client.V3Ref{"users": refs(userIDs)}
	_, err := s.client.V3(ctx, http.MethodPost, groupPath(id)+"/add-members", orgFilter(orgID), req, nil)
	return err
}

func (s *Service) RemoveMembers(ctx context.Context, orgID, id string, userIDs ...string) error {
	req := map[string][]client.V3Ref{"users": refs(userIDs)}
	_, err := s.client.V3(ctx, http.MethodPost, groupPath(id)+"/remove-members", orgFilter(orgID), req, nil)
	return err
}

// GrantProjects grants the group role on each project.
func (s *Service) GrantProjects(ctx context.Context, orgID, id, role string, projectIDs ...string) error {
	req := struct {
		Role     string         `json:"role"`
		Projects []client.V3Ref `json:"projects"`
	}{Role: role, Projects: refs(projectIDs)}
	_, err := s.client.V3(ctx, http.MethodPost, groupPath(id)+"/add-projects", orgFilter(orgID), req, nil)
	return err
}

// RevokeProjects revokes whatever role the group holds on each project.
func (s *Service) RevokeProjects(ctx context.Context, orgID, id string, projectIDs ...string) error {
	req := map[string][]client.V3Ref{"projects": refs(projectIDs)}
	_, err := s.client.V3(ctx, http.MethodPost, groupPath(id)+"/remove-projects", orgFilter(orgID), req, nil)
	return err
}

// ProjectGroups lists the groups that hold a role on projectID. Grants cannot
// be read individually, so this is how a grant is read back. The API supports
// it only for standalone organizations.
func (s *Service) ProjectGroups(ctx context.Context, orgID, projectID string) ([]ProjectGroup, error) {
	type attributes struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	q := orgFilter(orgID)
	q.Set("filter[project_id]", projectID)
	items, err := client.V3ListAll[client.V3Entity[attributes, struct{}]](
		ctx, s.client, "/groups", q,
	)
	if err != nil {
		return nil, err
	}
	groups := make([]ProjectGroup, 0, len(items))
	for _, item := range items {
		groups = append(groups, ProjectGroup{
			ID:   item.ID,
			Name: item.Attributes.Name,
			Role: item.Attributes.Role,
		})
	}
	return groups, nil
}
