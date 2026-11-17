// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package orgsettings_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
	"terraform-provider-circleci/internal/circleci/orgsettings"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	testTok      = "7b2d4f6a-8c0e-4a2b-9d4f-6a8c0e2b4d6f"
	unknownOrgID = "3e5a7c9b-1d3f-4b5d-8f7a-9c1e3b5d7f9a"
)

type fixture struct {
	fake  *fakecircle.Service
	url   string
	orgID string
	svc   *orgsettings.Service
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	fake := fakecircle.New(testTok)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	org, err := fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: "settings-org"})
	assert.NilError(t, err)
	return fixture{
		fake:  fake,
		url:   srv.URL + "/api/v2",
		orgID: org.ID.String(),
		svc:   orgsettings.NewService(client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")),
	}
}

// defaults are the managed settings a new org has on the fake, which are the
// values observed on a real organization.
func defaults() orgsettings.Settings {
	return orgsettings.Settings{
		EnableAIErrorSummarization:        common.Bool(true),
		EnableAIAgents:                    common.Bool(true),
		EnableMinorAIFeatures:             common.Bool(true),
		EnableUnversionedConfig:           common.Bool(false),
		EnableCertifiedPublicOrbs:         common.Bool(true),
		EnableUncertifiedPublicOrbs:       common.Bool(true),
		EnablePrivateOrbs:                 common.Bool(false),
		EnableChunkIPRanges:               common.Bool(false),
		EnableMarketingTracking:           common.Bool(true),
		EnablePreferencesTracking:         common.Bool(true),
		EnableStatisticsTracking:          common.Bool(true),
		IsPrivacyOptoutEnforced:           common.Bool(false),
		IsUserCheckoutKeysDisabled:        common.Bool(false),
		EnableImageBrownouts:              common.Bool(true),
		EnableResourceClassBrownouts:      common.Bool(true),
		IsContextGroupRestrictionRequired: common.Bool(false),
	}
}

func TestService_Settings(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)

	t.Run("get reports every managed setting", func(t *testing.T) {
		got, err := f.svc.Get(ctx, f.orgID)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(*got, defaults()))
	})

	t.Run("update sends only the settings it is given", func(t *testing.T) {
		got, err := f.svc.Update(ctx, f.orgID, orgsettings.Settings{
			EnablePrivateOrbs: common.Bool(true),
			EnableAIAgents:    common.Bool(false),
		})
		assert.NilError(t, err)

		want := defaults()
		want.EnablePrivateOrbs = common.Bool(true)
		want.EnableAIAgents = common.Bool(false)
		assert.Check(t, cmp.DeepEqual(*got, want))

		requests := f.fake.OrgSettingsRequests()
		assert.Check(t, cmp.DeepEqual(requests, []map[string]bool{
			{"enable_private_orbs": true, "enable_ai_agents": false},
		}))
	})

	t.Run("update with nothing to change only reads", func(t *testing.T) {
		got, err := f.svc.Update(ctx, f.orgID, orgsettings.Settings{})
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(*got.EnablePrivateOrbs, true))

		requests := f.fake.OrgSettingsRequests()
		assert.Check(t, cmp.Len(requests, 1))
	})
}

func TestService_SettingsErrors(t *testing.T) {
	ctx := context.TODO()
	f := newFixture(t)

	t.Run("get of an unknown org is not found", func(t *testing.T) {
		_, err := f.svc.Get(ctx, unknownOrgID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("update of an unknown org is not found", func(t *testing.T) {
		_, err := f.svc.Update(ctx, unknownOrgID, orgsettings.Settings{EnablePrivateOrbs: common.Bool(true)})
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("an invalid token is rejected", func(t *testing.T) {
		svc := orgsettings.NewService(client.NewClient(f.url, "not-the-token", "terraform-provider-circleci/test"))
		_, err := svc.Get(ctx, f.orgID)
		assert.Check(t, cmp.ErrorContains(err, "401 Unauthorized"))
	})

	t.Run("the fake rejects a setting it does not know", func(t *testing.T) {
		c := client.NewClient(f.url, testTok, "terraform-provider-circleci/test")
		body := map[string]any{"data": map[string]any{"attributes": map[string]any{"enable_everything": true}}}
		_, err := c.V3(ctx, http.MethodPost, "/orgs/"+f.orgID+"/update-settings", nil, body, nil)
		assert.Check(t, cmp.ErrorContains(err, "400 Bad Request"))
		assert.Check(t, cmp.ErrorContains(err, "unknown setting enable_everything"))
	})

	t.Run("no rejected write changed anything", func(t *testing.T) {
		requests := f.fake.OrgSettingsRequests()
		assert.Check(t, cmp.Len(requests, 0))
	})
}
