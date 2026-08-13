// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package project_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
	"terraform-provider-circleci/internal/circleci/project"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const testTok = "8f23dc1b-b7fd-4bed-9a2c-ec699b1ba810"

func TestProjectService_Get(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	ps := project.NewProjectService(c)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "test org",
	})
	assert.Assert(t, err)
	prj, err := fc.AddProject(fakecircle.NewProject{
		OrgID: org.ID,
		Name:  "test project",
	})
	assert.Assert(t, err)

	t.Run("get", func(t *testing.T) {
		ctx := context.TODO()
		gotProj, err := ps.Get(ctx, prj.Slug)
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(gotProj, &project.Project{
			Id:               prj.ID.String(),
			Name:             "test project",
			Slug:             prj.Slug,
			OrganizationName: "test org",
			OrganizationSlug: org.Slug,
			OrganizationId:   org.ID.String(),
			VcsInfo: common.VcsInfo{
				VcsUrl:        "git://github.com/dummy-value",
				Provider:      fakecircle.TypeCircleCI,
				DefaultBranch: "main",
			},
		}))
	})
}

func TestProjectService_Create(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	ps := project.NewProjectService(c)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "test org",
	})
	assert.Assert(t, err)

	var p *project.Project
	t.Run("create", func(t *testing.T) {
		ctx := context.TODO()
		var err error
		p, err = ps.Create(ctx, "test project name", org.ID.String())
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(p, &project.Project{
			Id:               "ignored",
			Name:             "test project name",
			Slug:             "ignored",
			OrganizationName: "test org",
			OrganizationSlug: org.Slug,
			OrganizationId:   org.ID.String(),
			VcsInfo: common.VcsInfo{
				VcsUrl:        "git://github.com/dummy-value",
				Provider:      fakecircle.TypeCircleCI,
				DefaultBranch: "main",
			},
		}, cmpopts.IgnoreFields(project.Project{}, "Id", "Slug")))
	})

	t.Run("get", func(t *testing.T) {
		p, err := fc.Project(uuid.MustParse(p.Id))
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(p, fakecircle.Project{
			ID:   p.ID,
			Name: "test project name",
			Slug: p.Slug,
			Org: fakecircle.Org{
				ID:   p.Org.ID,
				Type: fakecircle.TypeCircleCI,
				Name: "test org",
				Slug: p.Org.Slug,
			},
		}))
	})
}

func TestProjectService_Delete(t *testing.T) {
	ctx := context.TODO()
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	ps := project.NewProjectService(c)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "test org",
	})
	assert.Assert(t, err)
	prj, err := fc.AddProject(fakecircle.NewProject{
		OrgID: org.ID,
		Name:  "test project",
	})
	assert.Assert(t, err)

	t.Run("delete", func(t *testing.T) {
		err := ps.Delete(ctx, prj.Slug)
		assert.Assert(t, err)
	})

	t.Run("get", func(t *testing.T) {
		p, err := fc.Project(prj.ID)
		assert.Check(t, cmp.ErrorContains(err, "not found"))
		assert.Check(t, cmp.Equal(p.ID, uuid.Nil))
	})
}
