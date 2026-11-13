// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package pipeline_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
	"terraform-provider-circleci/internal/circleci/connection"
	"terraform-provider-circleci/internal/circleci/organization"
	"terraform-provider-circleci/internal/circleci/pipeline"
	"terraform-provider-circleci/internal/circleci/project"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	testTok             = "3a9c1e57-0b2d-4f6e-8a4c-6e2b0d8f1a3c"
	forgejoRepoFullName = "carta/service"
	forgejoRepoID       = "1234"
)

type fixture struct {
	fake      *fakecircle.Service
	client    *client.Client
	pipelines *pipeline.PipelineService
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	fake := fakecircle.New(testTok)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	return fixture{fake: fake, client: c, pipelines: pipeline.NewPipelineService(c)}
}

// forgejoProject returns a project that can hold Forgejo pipelines. connect is
// false to test what CircleCI does before the org is connected.
func (f fixture) forgejoProject(t *testing.T, connect bool) string {
	t.Helper()
	ctx := context.TODO()

	org, err := organization.NewOrganizationService(f.client).Create(ctx, "forgejo org", "circleci")
	assert.NilError(t, err)
	proj, err := project.NewProjectService(f.client).Create(ctx, "forgejo-project", org.Id)
	assert.NilError(t, err)

	if connect {
		_, err = connection.NewService(f.client).SetupForgejo(ctx, org.Id, connection.ForgejoSetup{
			URL:                "https://forgejo.example.com",
			Owner:              "carta",
			ConnectionAudience: "u:7:0f9e8d7c-6b5a-4c3d-2e1f-0a9b8c7d6e5f",
			CheckoutAudience:   "u:7:1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
		})
		assert.NilError(t, err)
	}
	f.fake.AddForgejoRepo(forgejoRepoFullName, forgejoRepoID)
	return proj.Id
}

func forgejoPipeline(repo common.Repo) pipeline.Pipeline {
	return pipeline.Pipeline{
		Name:           "build",
		Description:    "Builds the service",
		ConfigSource:   common.ConfigSource{Provider: pipeline.ProviderForgejo, Repo: repo, FilePath: ".circleci/config.yml"},
		CheckoutSource: common.CheckoutSource{Provider: pipeline.ProviderForgejo, Repo: repo},
	}
}

func TestPipelineService_Forgejo(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)
	projectID := f.forgejoProject(t, true)

	wantRepo := common.Repo{FullName: forgejoRepoFullName, ExternalId: forgejoRepoID}
	ignoreGenerated := cmpopts.IgnoreFields(pipeline.Pipeline{}, "ID", "CreatedAt")

	var created *pipeline.Pipeline
	t.Run("create from the repository's full name", func(t *testing.T) {
		var err error
		created, err = f.pipelines.Create(ctx, forgejoPipeline(common.Repo{FullName: forgejoRepoFullName}), projectID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(created, &pipeline.Pipeline{
			Name:           "build",
			Description:    "Builds the service",
			ConfigSource:   common.ConfigSource{Provider: "forgejo", Repo: wantRepo, FilePath: ".circleci/config.yml"},
			CheckoutSource: common.CheckoutSource{Provider: "forgejo", Repo: wantRepo},
		}, ignoreGenerated))
	})
	assert.Assert(t, created != nil)

	t.Run("get includes the repository", func(t *testing.T) {
		got, err := f.pipelines.Get(ctx, projectID, created.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, created))
	})

	t.Run("list includes the repository", func(t *testing.T) {
		got, err := f.pipelines.List(ctx, projectID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, []pipeline.Pipeline{*created}))
	})

	t.Run("update", func(t *testing.T) {
		changes := forgejoPipeline(wantRepo)
		changes.Description = "Builds and tests the service"
		changes.ConfigSource.FilePath = ".circleci/other.yml"
		got, err := f.pipelines.Update(ctx, changes, projectID, created.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got.Description, "Builds and tests the service"))
		assert.Check(t, cmp.Equal(got.ConfigSource.FilePath, ".circleci/other.yml"))
		assert.Check(t, cmp.DeepEqual(got.CheckoutSource.Repo, wantRepo))
	})

	t.Run("delete", func(t *testing.T) {
		err := f.pipelines.Delete(ctx, projectID, created.ID)
		assert.NilError(t, err)

		_, err = f.pipelines.Get(ctx, projectID, created.ID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})
}

func TestPipelineService_ForgejoRejected(t *testing.T) {
	ctx := context.TODO()

	t.Run("the org has no Forgejo connection", func(t *testing.T) {
		f := newFixture(t)
		projectID := f.forgejoProject(t, false)
		_, err := f.pipelines.Create(ctx, forgejoPipeline(common.Repo{FullName: forgejoRepoFullName}), projectID)
		assert.Check(t, cmp.ErrorContains(err, "400 Bad Request"))
		assert.Check(t, cmp.ErrorContains(err, "no forgejo connection"))
	})

	t.Run("the repository is not visible to the connection", func(t *testing.T) {
		f := newFixture(t)
		projectID := f.forgejoProject(t, true)
		_, err := f.pipelines.Create(ctx, forgejoPipeline(common.Repo{FullName: "carta/missing"}), projectID)
		assert.Check(t, cmp.ErrorContains(err, "400 Bad Request"))
		assert.Check(t, cmp.ErrorContains(err, "repository not found"))
	})
}

func TestPipelineService_GithubStaysOnV2(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)
	const projectID = "0b8e2f4a-6c1d-4e3b-9a7f-5d2c8e1b4a6f"
	repo := common.Repo{ExternalId: "964220967"}

	created, err := f.pipelines.Create(ctx, pipeline.Pipeline{
		Name:           "build",
		ConfigSource:   common.ConfigSource{Provider: "github_app", Repo: repo, FilePath: ".circleci/config.yml"},
		CheckoutSource: common.CheckoutSource{Provider: "github_app", Repo: repo},
	}, projectID)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(created.CheckoutSource.Repo.FullName, "fake-org/repo-964220967"))

	updated, err := f.pipelines.Update(ctx, pipeline.Pipeline{
		Name:           "build",
		Description:    "changed",
		ConfigSource:   common.ConfigSource{FilePath: ".circleci/config.yml"},
		CheckoutSource: common.CheckoutSource{Provider: "github_app", Repo: common.Repo{ExternalId: "111"}},
	}, projectID, created.ID)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(updated.Description, "changed"))
	assert.Check(t, cmp.Equal(updated.CheckoutSource.Repo.ExternalId, "111"))

	got, err := f.pipelines.Get(ctx, projectID, created.ID)
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(got, updated))
}
