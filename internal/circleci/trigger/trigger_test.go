// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package trigger_test

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
	"terraform-provider-circleci/internal/circleci/trigger"
)

const (
	testTok             = "8d2f6a1c-3e5b-4c7d-9f0a-1b2c3d4e5f6a"
	forgejoRepoFullName = "carta/service"
	forgejoRepoID       = "1234"
)

// forgejoPipeline creates a standalone org connected to Forgejo, a project in
// it and a Forgejo pipeline, returning the project and pipeline IDs.
func forgejoPipeline(t *testing.T, fake *fakecircle.Service, c *client.Client) (string, string) {
	t.Helper()
	ctx := context.TODO()

	org, err := organization.NewOrganizationService(c).Create(ctx, "forgejo org", "circleci")
	assert.NilError(t, err)
	proj, err := project.NewProjectService(c).Create(ctx, "forgejo-project", org.Id)
	assert.NilError(t, err)
	_, err = connection.NewService(c).SetupForgejo(ctx, org.Id, connection.ForgejoSetup{
		URL:                "https://forgejo.example.com",
		Owner:              "carta",
		ConnectionAudience: "u:7:0f9e8d7c-6b5a-4c3d-2e1f-0a9b8c7d6e5f",
		CheckoutAudience:   "u:7:1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
	})
	assert.NilError(t, err)
	fake.AddForgejoRepo(forgejoRepoFullName, forgejoRepoID)

	repo := common.Repo{ExternalId: forgejoRepoID}
	p, err := pipeline.NewPipelineService(c).Create(ctx, pipeline.Pipeline{
		Name:           "build",
		ConfigSource:   common.ConfigSource{Provider: "forgejo", Repo: repo, FilePath: ".circleci/config.yml"},
		CheckoutSource: common.CheckoutSource{Provider: "forgejo", Repo: repo},
	}, proj.Id)
	assert.NilError(t, err)
	return proj.Id, p.ID
}

func newClient(t *testing.T) (*fakecircle.Service, *client.Client) {
	t.Helper()
	fake := fakecircle.New(testTok)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return fake, client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
}

func TestTriggerService_Forgejo(t *testing.T) {
	ctx := context.TODO()
	fake, c := newClient(t)
	svc := trigger.NewTriggerService(c)
	projectID, pipelineID := forgejoPipeline(t, fake, c)
	disabled := false

	var created *trigger.TriggerResponse
	t.Run("create from the repository's full name", func(t *testing.T) {
		var err error
		created, err = svc.Create(ctx, trigger.Trigger{
			EventSource: common.EventSource{Provider: "forgejo", Repo: common.Repo{FullName: forgejoRepoFullName}},
			EventPreset: "only-open-prs",
			Disabled:    &disabled,
		}, projectID, pipelineID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(created, &trigger.TriggerResponse{
			EventSource: common.EventSourceResponse{
				Provider: "forgejo",
				Repo:     common.Repo{FullName: forgejoRepoFullName, ExternalId: forgejoRepoID},
			},
			EventPreset: "only-open-prs",
			Disabled:    &disabled,
			PipelineID:  pipelineID,
		}, cmpopts.IgnoreFields(trigger.TriggerResponse{}, "ID", "CreatedAt")))
	})
	assert.Assert(t, created != nil)

	t.Run("get includes the repository", func(t *testing.T) {
		got, err := svc.Get(ctx, projectID, created.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, created))
	})

	t.Run("list includes the repository", func(t *testing.T) {
		got, err := svc.List(ctx, projectID, pipelineID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, []trigger.TriggerResponse{*created}))
	})

	t.Run("update", func(t *testing.T) {
		nowDisabled := true
		got, err := svc.Update(ctx, trigger.Trigger{
			EventSource: common.EventSource{Provider: "forgejo"},
			EventPreset: "only-merged-prs",
			Disabled:    &nowDisabled,
		}, projectID, created.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got.EventPreset, "only-merged-prs"))
		assert.Check(t, cmp.DeepEqual(got.Disabled, &nowDisabled))
		assert.Check(t, cmp.Equal(got.EventSource.Repo.ExternalId, forgejoRepoID))
	})

	t.Run("delete", func(t *testing.T) {
		err := svc.Delete(ctx, projectID, created.ID)
		assert.NilError(t, err)

		_, err = svc.Get(ctx, projectID, created.ID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})
}

func TestTriggerService_ForgejoRejected(t *testing.T) {
	ctx := context.TODO()
	fake, c := newClient(t)
	svc := trigger.NewTriggerService(c)
	projectID, pipelineID := forgejoPipeline(t, fake, c)

	tests := []struct {
		name    string
		trigger trigger.Trigger
		wantErr string
	}{
		{
			name: "a GitHub-only preset",
			trigger: trigger.Trigger{
				EventSource: common.EventSource{Provider: "forgejo", Repo: common.Repo{ExternalId: forgejoRepoID}},
				EventPreset: "only-labeled-prs",
			},
			wantErr: "unknown event key for forgejo",
		},
		{
			name: "a repository the connection cannot see",
			trigger: trigger.Trigger{
				EventSource: common.EventSource{Provider: "forgejo", Repo: common.Repo{FullName: "carta/missing"}},
				EventPreset: "all-pushes",
			},
			wantErr: "event repository not found",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tc.trigger, projectID, pipelineID)
			assert.Check(t, cmp.ErrorContains(err, "400 Bad Request"))
			assert.Check(t, cmp.ErrorContains(err, tc.wantErr))
		})
	}
}
