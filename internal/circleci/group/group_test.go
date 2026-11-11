// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package group_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/group"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	testTok   = "4c8e2a6f-1b3d-4e5f-a7b9-c1d3e5f7a9b1"
	orgID     = "9e1c3a5b-7d2f-4a6e-8c0b-2d4f6a8c0e2a"
	otherOrg  = "1f3b5d7a-9c2e-4b6d-8f0a-3c5e7a9b1d3f"
	userID    = "2a4c6e8a-0b1d-4f3a-9c5e-7a9b1c3d5e7f"
	projectID = "6b8d0f2a-4c6e-4a8b-9d1f-3a5c7e9b1d3f"
)

func newService(t *testing.T) *group.Service {
	t.Helper()
	srv := httptest.NewServer(fakecircle.New(testTok))
	t.Cleanup(srv.Close)
	return group.NewService(client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test"))
}

func TestService_Group(t *testing.T) {
	ctx := context.TODO()
	svc := newService(t)

	var g *group.Group
	t.Run("create", func(t *testing.T) {
		var err error
		g, err = svc.Create(ctx, orgID, "platform", "The platform team")
		assert.NilError(t, err)
		assert.Check(t, g.ID != "")
		assert.Check(t, cmp.DeepEqual(g, &group.Group{
			ID:          g.ID,
			OrgID:       orgID,
			Name:        "platform",
			Description: "The platform team",
		}))
	})
	assert.Assert(t, g != nil)

	t.Run("update", func(t *testing.T) {
		got, err := svc.Update(ctx, orgID, g.ID, "platform-eng", "")
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got.Name, "platform-eng"))
		assert.Check(t, cmp.Equal(got.Description, ""))
	})

	t.Run("a group is not visible from another org", func(t *testing.T) {
		_, err := svc.Get(ctx, otherOrg, g.ID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("members", func(t *testing.T) {
		err := svc.AddMembers(ctx, orgID, g.ID, userID)
		assert.NilError(t, err)

		members, err := svc.Members(ctx, orgID, g.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(members, []group.Member{{UserID: userID}}))

		err = svc.RemoveMembers(ctx, orgID, g.ID, userID)
		assert.NilError(t, err)

		members, err = svc.Members(ctx, orgID, g.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(members, 0))
	})

	t.Run("project roles", func(t *testing.T) {
		err := svc.GrantProjects(ctx, orgID, g.ID, "project-viewer", projectID)
		assert.NilError(t, err)

		groups, err := svc.ProjectGroups(ctx, orgID, projectID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(groups, []group.ProjectGroup{
			{ID: g.ID, Name: "platform-eng", Role: "project-viewer"},
		}))

		err = svc.RevokeProjects(ctx, orgID, g.ID, projectID)
		assert.NilError(t, err)

		groups, err = svc.ProjectGroups(ctx, orgID, projectID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(groups, 0))
	})

	t.Run("an unknown role is rejected", func(t *testing.T) {
		err := svc.GrantProjects(ctx, orgID, g.ID, "project-owner", projectID)
		assert.Check(t, cmp.ErrorContains(err, "400 Bad Request"))
	})

	t.Run("delete", func(t *testing.T) {
		err := svc.Delete(ctx, orgID, g.ID)
		assert.NilError(t, err)

		_, err = svc.Get(ctx, orgID, g.ID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})
}
