// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package project_test

import (
	"context"
	"net/http/httptest"
	"strings"
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

func TestProjectService_Settings(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	ps := project.NewProjectService(c)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "test org",
	})
	assert.NilError(t, err)

	openProj, err := fc.AddProject(fakecircle.NewProject{
		OrgID: org.ID,
		Name:  "open project",
	})
	assert.NilError(t, err)

	closed := false
	closedProj, err := fc.AddProject(fakecircle.NewProject{
		OrgID:          org.ID,
		Name:           "closed project",
		RepoOpenSource: &closed,
	})
	assert.NilError(t, err)

	t.Run("get defaults", func(t *testing.T) {
		provider, organization, name := projectSlugParts(t, openProj.Slug)
		got, err := ps.GetSettings(t.Context(), provider, organization, name)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, defaultProjectSettings()))
	})

	t.Run("update oss", func(t *testing.T) {
		provider, organization, name := projectSlugParts(t, openProj.Slug)
		want := defaultProjectSettings()
		want.Advanced.OSS = common.Bool(true)
		want.Advanced.AutocancelBuilds = common.Bool(true)
		want.Advanced.PROnlyBranchOverrides = []string{"main"}

		got, err := ps.UpdateSettings(t.Context(), project.ProjectSettings{
			Advanced: project.AdvanceSettings{
				AutocancelBuilds:      common.Bool(true),
				OSS:                   common.Bool(true),
				PROnlyBranchOverrides: []string{"main"},
			},
		}, provider, organization, name)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, want))

		t.Run("get", func(t *testing.T) {
			read, err := ps.GetSettings(t.Context(), provider, organization, name)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(read, want))
		})
	})

	t.Run("oss false is kept when the repository is not open source", func(t *testing.T) {
		provider, organization, name := projectSlugParts(t, closedProj.Slug)
		got, err := ps.UpdateSettings(t.Context(), project.ProjectSettings{
			Advanced: project.AdvanceSettings{
				AutocancelBuilds: common.Bool(true),
				OSS:              common.Bool(false),
			},
		}, provider, organization, name)
		assert.NilError(t, err)

		want := defaultProjectSettings()
		want.Advanced.AutocancelBuilds = common.Bool(true)
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("oss true is ignored when the repository is not open source", func(t *testing.T) {
		provider, organization, name := projectSlugParts(t, closedProj.Slug)
		got, err := ps.UpdateSettings(t.Context(), project.ProjectSettings{
			Advanced: project.AdvanceSettings{
				AutocancelBuilds: common.Bool(false),
				BuildForkPrs:     common.Bool(true),
				OSS:              common.Bool(true),
			},
		}, provider, organization, name)
		assert.NilError(t, err)

		want := defaultProjectSettings()
		want.Advanced.BuildForkPrs = common.Bool(true)
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("get missing project", func(t *testing.T) {
		got, err := ps.GetSettings(t.Context(), "circleci", "missing-org", "missing-project")
		assert.Check(t, cmp.ErrorContains(err, "project not found"))
		assert.Check(t, cmp.Nil(got))
	})

	t.Run("update missing project", func(t *testing.T) {
		got, err := ps.UpdateSettings(t.Context(), project.ProjectSettings{
			Advanced: project.AdvanceSettings{
				OSS: common.Bool(true),
			},
		}, "circleci", "missing-org", "missing-project")
		assert.Check(t, cmp.ErrorContains(err, "project not found"))
		assert.Check(t, cmp.Nil(got))
	})

	t.Run("get rejects unknown provider", func(t *testing.T) {
		got, err := ps.GetSettings(t.Context(), "gitlab", "org", "project")
		assert.Check(t, cmp.ErrorContains(err, "invalid org type"))
		assert.Check(t, cmp.Nil(got))
	})
}

func projectSlugParts(t *testing.T, slug string) (string, string, string) {
	t.Helper()

	parts := strings.Split(slug, "/")
	assert.Assert(t, cmp.Len(parts, 3))
	return parts[0], parts[1], parts[2]
}

func defaultProjectSettings() *project.ProjectSettings {
	return &project.ProjectSettings{
		Advanced: project.AdvanceSettings{
			AutocancelBuilds:           common.Bool(false),
			BuildForkPrs:               common.Bool(false),
			DisableSSH:                 common.Bool(false),
			ForksReceiveSecretEnvVars:  common.Bool(true),
			OSS:                        common.Bool(false),
			SetGithubStatus:            common.Bool(false),
			SetupWorkflows:             common.Bool(false),
			WriteSettingsRequiresAdmin: common.Bool(false),
			PROnlyBranchOverrides:      []string{},
		},
	}
}
