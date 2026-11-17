// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package notification_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/notification"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	testTok = "4c8e2a6f-1b3d-4e5f-a7b9-c1d3e5f7a9b1"

	prefDefaultBranch = "c9930e26-d1e1-46e2-a5d2-45c77843fcd2"
	prefApproval      = "3b5d94e8-6feb-4eb6-8458-ae2571fb4192"
	prefSucceeded     = "796a37f9-7b11-4e92-9322-927e76583cf3"
	prefFailed        = "4234f1ce-9bcd-4a9c-966c-4e365c69e1ed"
	prefInconclusive  = "74d43e43-955f-4596-bda6-c0ef96ba3119"
)

type fixture struct {
	fake      *fakecircle.Service
	client    *client.Client
	svc       *notification.Service
	orgID     string
	projectID string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	fake := fakecircle.New(testTok)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	org, err := fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: "carta"})
	assert.NilError(t, err)
	p, err := fake.AddProject(fakecircle.NewProject{OrgID: org.ID, Name: "service"})
	assert.NilError(t, err)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	return fixture{
		fake:      fake,
		client:    c,
		svc:       notification.NewService(c),
		orgID:     org.ID.String(),
		projectID: p.ID.String(),
	}
}

func TestService_ChannelConfig(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)
	f.fake.AddSlackChannel("C0B8D01J6RF", "builds")

	var created *notification.ChannelConfig
	t.Run("create", func(t *testing.T) {
		var err error
		created, err = f.svc.CreateSlackChannel(ctx, f.orgID, f.projectID, "C0B8D01J6RF", true)
		assert.NilError(t, err)
		assert.Check(t, created.ID != "")
		name := "builds"
		assert.Check(t, cmp.DeepEqual(created, &notification.ChannelConfig{
			ID:          created.ID,
			OrgID:       f.orgID,
			ProjectID:   f.projectID,
			ChannelType: "slack",
			Target:      "C0B8D01J6RF",
			Enabled:     true,
			ChannelName: &name,
		}))
	})
	assert.Assert(t, created != nil)

	t.Run("get", func(t *testing.T) {
		got, err := f.svc.GetChannelConfig(ctx, created.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, created))
	})

	t.Run("list the project's configs", func(t *testing.T) {
		got, err := f.svc.ListProjectChannelConfigs(ctx, f.orgID, f.projectID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, []notification.ChannelConfig{*created}))
	})

	t.Run("a second config for the project is rejected", func(t *testing.T) {
		_, err := f.svc.CreateSlackChannel(ctx, f.orgID, f.projectID, "C0OTHER0001", true)
		assert.Check(t, cmp.ErrorContains(err, "409"))
	})

	t.Run("update", func(t *testing.T) {
		got, err := f.svc.UpdateChannelConfig(ctx, created.ID, "C0NEW000001", false)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got.Target, "C0NEW000001"))
		assert.Check(t, !got.Enabled, "the config should be disabled")
		assert.Check(t, cmp.Nil(got.ChannelName))
	})

	t.Run("delete", func(t *testing.T) {
		err := f.svc.DeleteChannelConfig(ctx, created.ID)
		assert.NilError(t, err)

		_, err = f.svc.GetChannelConfig(ctx, created.ID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)

		configs, err := f.svc.ListProjectChannelConfigs(ctx, f.orgID, f.projectID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(configs, 0))
	})
}

func TestService_ChannelConfigErrors(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)
	missing := uuid.NewString()

	t.Run("create for an unknown project is not found", func(t *testing.T) {
		_, err := f.svc.CreateSlackChannel(ctx, f.orgID, missing, "C0B8D01J6RF", true)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("get of an unknown config is not found", func(t *testing.T) {
		_, err := f.svc.GetChannelConfig(ctx, missing)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("update of an unknown config is not found", func(t *testing.T) {
		_, err := f.svc.UpdateChannelConfig(ctx, missing, "C0B8D01J6RF", true)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("delete of an unknown config is not found", func(t *testing.T) {
		err := f.svc.DeleteChannelConfig(ctx, missing)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("the fake rejects an update body wrapped in data", func(t *testing.T) {
		created, err := f.svc.CreateSlackChannel(ctx, f.orgID, f.projectID, "C0B8D01J6RF", true)
		assert.NilError(t, err)
		body := map[string]any{"data": map[string]any{"attributes": map[string]any{"target": "C0B8D01J6RF"}}}
		_, err = f.client.V3(ctx, http.MethodPost, "/notification/channel-configs/"+created.ID+"/update", nil, body, nil)
		assert.Check(t, cmp.ErrorContains(err, "400"))
	})
}

func TestService_WelcomeMessages(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)

	var id string
	t.Run("creating posts a welcome message", func(t *testing.T) {
		created, err := f.svc.CreateSlackChannel(ctx, f.orgID, f.projectID, "C0B8D01J6RF", true)
		assert.NilError(t, err)
		id = created.ID
		count := f.fake.SlackWelcomeMessages()
		assert.Check(t, cmp.Equal(count, 1))
	})

	t.Run("an update that leaves it enabled posts again, even without a change", func(t *testing.T) {
		_, err := f.svc.UpdateChannelConfig(ctx, id, "C0B8D01J6RF", true)
		assert.NilError(t, err)
		count := f.fake.SlackWelcomeMessages()
		assert.Check(t, cmp.Equal(count, 2))
	})

	t.Run("disabling does not post", func(t *testing.T) {
		_, err := f.svc.UpdateChannelConfig(ctx, id, "C0B8D01J6RF", false)
		assert.NilError(t, err)
		count := f.fake.SlackWelcomeMessages()
		assert.Check(t, cmp.Equal(count, 2))
	})
}

func enabledIDs(prefs []notification.Preference) map[string]bool {
	out := map[string]bool{}
	for _, p := range prefs {
		out[p.ID] = p.Enabled
	}
	return out
}

func TestService_Preferences(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)

	t.Run("list follows every page", func(t *testing.T) {
		prefs, err := f.svc.ListProjectPreferences(ctx, f.orgID, f.projectID)
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(prefs, 6))
		assert.Check(t, cmp.DeepEqual(prefs[1], notification.Preference{
			ID:               prefDefaultBranch,
			Name:             "On default branch (failed, error, unauthorized)",
			Channel:          "slack",
			GroupName:        "Workflow Completed",
			SectionName:      "Build",
			UserConfigurable: true,
			Enabled:          true,
		}))
		assert.Check(t, cmp.Equal(prefs[0].ID, fakecircle.NonConfigurablePreferenceID))
		assert.Check(t, !prefs[0].UserConfigurable, "the seeded non-configurable preference")
	})

	t.Run("bulk update", func(t *testing.T) {
		err := f.svc.UpdateProjectPreferences(ctx, f.orgID, f.projectID, map[string]bool{
			prefDefaultBranch: false,
			prefApproval:      true,
			prefFailed:        true,
		})
		assert.NilError(t, err)

		prefs, err := f.svc.ListProjectPreferences(ctx, f.orgID, f.projectID)
		assert.NilError(t, err)
		got := enabledIDs(prefs)
		assert.Check(t, cmp.DeepEqual(got, map[string]bool{
			fakecircle.NonConfigurablePreferenceID: true,
			prefDefaultBranch:                      false,
			prefApproval:                           true,
			prefSucceeded:                          false,
			prefFailed:                             true,
			prefInconclusive:                       false,
		}))
	})
}

func TestService_PreferencesErrors(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)

	t.Run("listing an unknown project is not found", func(t *testing.T) {
		_, err := f.svc.ListProjectPreferences(ctx, f.orgID, uuid.NewString())
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("updating an unknown preference is rejected", func(t *testing.T) {
		err := f.svc.UpdateProjectPreferences(ctx, f.orgID, f.projectID, map[string]bool{uuid.NewString(): true})
		assert.Check(t, cmp.ErrorContains(err, "400"))
	})

	t.Run("updating a preference that is not user configurable is rejected", func(t *testing.T) {
		err := f.svc.UpdateProjectPreferences(ctx, f.orgID, f.projectID, map[string]bool{
			fakecircle.NonConfigurablePreferenceID: false,
		})
		assert.Check(t, cmp.ErrorContains(err, "400"))
	})

	t.Run("an empty update sends nothing", func(t *testing.T) {
		err := f.svc.UpdateProjectPreferences(ctx, f.orgID, f.projectID, map[string]bool{})
		assert.NilError(t, err)
	})
}
