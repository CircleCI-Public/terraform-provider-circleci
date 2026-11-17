// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"maps"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
	"terraform-provider-circleci/internal/circleci/orgsettings"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	fakeOrgSettingsToken   = "fake-org-settings-token"
	orgSettingsAddress     = "circleci_organization_settings.this"
	orgSettingsUnknownOrg  = "8d0f2b4c-6e8a-4c0d-9f2b-4d6f8a0c2e4b"
	orgSettingsFakeOrgName = "settings-org"
)

type fakeOrgSettings struct {
	fake  *fakecircle.Service
	host  string
	orgID string
}

func newFakeOrgSettings(t *testing.T) fakeOrgSettings {
	t.Helper()
	fake := fakecircle.New(fakeOrgSettingsToken)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	org, err := fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: orgSettingsFakeOrgName})
	assert.NilError(t, err)
	return fakeOrgSettings{fake: fake, host: srv.URL + "/api/v2", orgID: org.ID.String()}
}

// service is an API client for changing settings outside Terraform.
func (f fakeOrgSettings) service() *orgsettings.Service {
	return orgsettings.NewService(client.NewClient(f.host, fakeOrgSettingsToken, "terraform-provider-circleci/test"))
}

func testAccOrgSettingsConfig(host, orgID, settings string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_organization_settings" "this" {
  organization_id = %[3]q
%[4]s
}
`, host, fakeOrgSettingsToken, orgID, settings)
}

// expectRequests checks the attributes of every update-settings request the
// fake has applied so far, so a test can tell a setting that was left out
// from one written with its current value.
func expectRequests(f fakeOrgSettings, want ...map[string]bool) func(*terraform.State) error {
	return func(*terraform.State) error {
		got := f.fake.OrgSettingsRequests()
		if len(got) != len(want) {
			return fmt.Errorf("got %d update-settings requests %v, want %d %v", len(got), got, len(want), want)
		}
		for i := range want {
			if !maps.Equal(got[i], want[i]) {
				return fmt.Errorf("update-settings request %d carried %v, want %v", i, got[i], want[i])
			}
		}
		return nil
	}
}

func expectBool(attr string, want bool) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(orgSettingsAddress, tfjsonpath.New(attr), knownvalue.Bool(want))
}

func TestAccOrganizationSettings(t *testing.T) {
	f := newFakeOrgSettings(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create writes only what the configuration sets. Every other
				// setting keeps its current value and reads back without a diff,
				// which the post-apply empty plan check enforces.
				Config: testAccOrgSettingsConfig(f.host, f.orgID, `
  enable_private_orbs = true
  enable_ai_agents    = false
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(orgSettingsAddress, tfjsonpath.New("id"), knownvalue.StringExact(f.orgID)),
					expectBool("enable_private_orbs", true),
					expectBool("enable_ai_agents", false),
					expectBool("enable_certified_public_orbs", true),
					expectBool("enable_uncertified_public_orbs", true),
					expectBool("is_context_group_restriction_required", false),
				},
				Check: expectRequests(f,
					map[string]bool{"enable_private_orbs": true, "enable_ai_agents": false},
				),
			},
			{
				Config: testAccOrgSettingsConfig(f.host, f.orgID, `
  enable_private_orbs                   = false
  enable_ai_agents                      = false
  is_context_group_restriction_required = true
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(orgSettingsAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					expectBool("enable_private_orbs", false),
					expectBool("enable_ai_agents", false),
					expectBool("is_context_group_restriction_required", true),
					expectBool("enable_certified_public_orbs", true),
				},
				Check: expectRequests(f,
					map[string]bool{"enable_private_orbs": true, "enable_ai_agents": false},
					map[string]bool{
						"enable_private_orbs":                   false,
						"enable_ai_agents":                      false,
						"is_context_group_restriction_required": true,
					},
				),
			},
			{
				ResourceName:      orgSettingsAddress,
				ImportState:       true,
				ImportStateId:     f.orgID,
				ImportStateVerify: true,
			},
		},
		// Settings cannot be deleted, so destroying the resource only stops
		// managing them: no write, and the org keeps the applied values.
		CheckDestroy: func(s *terraform.State) error {
			if err := expectRequests(f,
				map[string]bool{"enable_private_orbs": true, "enable_ai_agents": false},
				map[string]bool{
					"enable_private_orbs":                   false,
					"enable_ai_agents":                      false,
					"is_context_group_restriction_required": true,
				},
			)(s); err != nil {
				return err
			}
			orgID, err := uuid.Parse(f.orgID)
			if err != nil {
				return err
			}
			settings := f.fake.OrgSettings(orgID)
			if !settings["is_context_group_restriction_required"] || settings["enable_private_orbs"] || settings["enable_ai_agents"] {
				return fmt.Errorf("destroy changed the organization's settings: %v", settings)
			}
			return nil
		},
	})
}

func TestAccOrganizationSettingsNothingConfigured(t *testing.T) {
	f := newFakeOrgSettings(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrgSettingsConfig(f.host, f.orgID, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					expectBool("enable_private_orbs", false),
					expectBool("enable_ai_agents", true),
					expectBool("enable_resource_class_brownouts", true),
				},
				Check: expectRequests(f),
			},
		},
	})
}

func TestAccOrganizationSettingsDrift(t *testing.T) {
	f := newFakeOrgSettings(t)
	config := testAccOrgSettingsConfig(f.host, f.orgID, `
  enable_private_orbs = true
`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  expectRequests(f, map[string]bool{"enable_private_orbs": true}),
			},
			{
				// A managed setting changed outside Terraform is put back. An
				// unmanaged one changed at the same time is left as it is.
				PreConfig: func() {
					_, err := f.service().Update(context.TODO(), f.orgID, orgsettings.Settings{
						EnablePrivateOrbs: common.Bool(false),
						EnableAIAgents:    common.Bool(false),
					})
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(orgSettingsAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(orgSettingsAddress, tfjsonpath.New("enable_private_orbs"), knownvalue.Bool(true)),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					expectBool("enable_private_orbs", true),
					expectBool("enable_ai_agents", false),
				},
				Check: expectRequests(f,
					map[string]bool{"enable_private_orbs": true},
					map[string]bool{"enable_private_orbs": false, "enable_ai_agents": false},
					map[string]bool{"enable_private_orbs": true},
				),
			},
			{
				// A change to an unmanaged setting alone is not a diff.
				PreConfig: func() {
					_, err := f.service().Update(context.TODO(), f.orgID, orgsettings.Settings{
						EnableAIAgents: common.Bool(true),
					})
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					expectBool("enable_ai_agents", true),
				},
			},
		},
	})
}

func TestAccOrganizationSettingsUnknownOrg(t *testing.T) {
	f := newFakeOrgSettings(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccOrgSettingsConfig(f.host, orgSettingsUnknownOrg, "enable_private_orbs = true"),
				ExpectError: regexp.MustCompile(`(?s)Error reading CircleCI organization settings.*organization admin`),
			},
		},
	})
}

func TestAccOrganizationSettingsExcludedSettings(t *testing.T) {
	f := newFakeOrgSettings(t)

	for _, attr := range []string{
		"is_runner_terms_of_service_accepted",
		"is_running_disabled",
		"is_code_signing_restricted_to_admins",
	} {
		t.Run(attr, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccOrgSettingsConfig(f.host, f.orgID, attr+" = false"),
						ExpectError: regexp.MustCompile(`(?s)An argument named "` + attr + `" is not expected\s+here`),
					},
				},
			})
		})
	}
}
