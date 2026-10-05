// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/common"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const fakeProjectToken = "fake-project-token"

func TestOSSEnableIgnored(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		requested types.Bool
		applied   *bool
		want      bool
	}{
		{
			name:      "unset",
			requested: types.BoolNull(),
			want:      false,
		},
		{
			name:      "unknown",
			requested: types.BoolUnknown(),
			want:      false,
		},
		{
			name:      "explicit false",
			requested: types.BoolValue(false),
			applied:   common.Bool(false),
			want:      false,
		},
		{
			name:      "applied",
			requested: types.BoolValue(true),
			applied:   common.Bool(true),
			want:      false,
		},
		{
			name:      "left false",
			requested: types.BoolValue(true),
			applied:   common.Bool(false),
			want:      true,
		},
		{
			name:      "missing from response",
			requested: types.BoolValue(true),
			want:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ossEnableIgnored(tc.requested, tc.applied)
			assert.Check(t, cmp.Equal(got, tc.want))
		})
	}
}

func TestAccProjectResourceOSS(t *testing.T) {
	fc := fakecircle.New(fakeProjectToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "oss org",
	})
	assert.NilError(t, err)

	host := srv.URL + "/api/v2"
	orgID := org.ID.String()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceOSSConfig(host, orgID, "oss-project", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact("oss-project"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgID),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("oss"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("forks_receive_secret_env_vars"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: testAccProjectResourceOSSConfig(host, orgID, "oss-project", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"circleci_project.test_project",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("oss"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("forks_receive_secret_env_vars"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				ResourceName:      "circleci_project.test_project",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: projectImportSlug,
			},
		},
	})
}

// TestAccProjectResourceOSSFalseWhenNotOpenSource covers oss = false on a
// project CircleCI will not let the flag be set on. The practitioner's
// configuration has to apply, be importable, and leave the other advanced
// settings alone.
func TestAccProjectResourceOSSFalseWhenNotOpenSource(t *testing.T) {
	fc := fakecircle.New(fakeProjectToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "closed org false",
	})
	assert.NilError(t, err)

	config := testAccProjectResourceOSSConfig(srv.URL+"/api/v2", org.ID.String(), "closed-source-false", false)
	ossFalse := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(
			"circleci_project.test_project",
			tfjsonpath.New("oss"),
			knownvalue.Bool(false),
		),
		statecheck.ExpectKnownValue(
			"circleci_project.test_project",
			tfjsonpath.New("forks_receive_secret_env_vars"),
			knownvalue.Bool(true),
		),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ConfigStateChecks: ossFalse,
			},
			{
				Config:            config,
				ConfigStateChecks: ossFalse,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ResourceName:      "circleci_project.test_project",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: projectImportSlug,
			},
		},
	})

	t.Run("the feature flag is never written", func(t *testing.T) {
		writes := fc.OSSWrites()
		assert.Check(t, cmp.Equal(writes, 0))
	})
}

func TestAccProjectResourceOSSNotOpenSource(t *testing.T) {
	fc := fakecircle.New(fakeProjectToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "closed org",
	})
	assert.NilError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectResourceOSSConfig(srv.URL+"/api/v2", org.ID.String(), "closed-source", true),
				ExpectError: regexp.MustCompile(`did not enable open source builds`),
			},
		},
	})
}

// TestAccProjectResourceCreateSettings pins the advanced settings a create
// sends. forks_receive_secret_env_vars decides whether a forked pull request
// can read the project's secrets, so a configuration that never mentions it
// must not have a value chosen on its behalf.
func TestAccProjectResourceCreateSettings(t *testing.T) {
	t.Run("settings the configuration leaves out are omitted", func(t *testing.T) {
		// The fake seeds forks_receive_secret_env_vars on, so a create that
		// leaves the setting alone reads back the project's own value and one
		// that writes false over it does not.
		sent := createProjectSettings(t, "create-unset", "", statecheck.ExpectKnownValue(
			"circleci_project.test_project",
			tfjsonpath.New("forks_receive_secret_env_vars"),
			knownvalue.Bool(true),
		))
		assert.Check(t, cmp.DeepEqual(sent, map[string]any{
			"build_fork_prs":    false,
			"disable_ssh":       false,
			"set_github_status": false,
		}))
	})

	t.Run("settings the configuration sets are sent", func(t *testing.T) {
		sent := createProjectSettings(t, "create-set", `
  auto_cancel_builds            = true
  forks_receive_secret_env_vars = false`, statecheck.ExpectKnownValue(
			"circleci_project.test_project",
			tfjsonpath.New("forks_receive_secret_env_vars"),
			knownvalue.Bool(false),
		))
		assert.Check(t, cmp.DeepEqual(sent, map[string]any{
			"autocancel_builds":             true,
			"build_fork_prs":                false,
			"disable_ssh":                   false,
			"forks_receive_secret_env_vars": false,
			"set_github_status":             false,
		}))
	})
}

// createProjectSettings creates a project against a fresh fake and returns the
// advanced settings object the create sent, so that a field the request left
// out can be told apart from one it sent as false.
func createProjectSettings(t *testing.T, name, settings string, checks ...statecheck.StateCheck) map[string]any {
	t.Helper()

	fc := fakecircle.New(fakeProjectToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "create settings org",
	})
	assert.NilError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            testAccProjectResourceSettingsConfig(srv.URL+"/api/v2", org.ID.String(), name, settings),
				ConfigStateChecks: checks,
			},
		},
	})

	requests := fc.SettingsRequests()
	assert.Assert(t, cmp.Len(requests, 1))
	return requests[0]
}

func testAccProjectResourceSettingsConfig(host, orgID, name, settings string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_project" "test_project" {
  name            = %[3]q
  organization_id = %[4]q%[5]s
}
`, host, fakeProjectToken, name, orgID, settings)
}

func testAccProjectResourceOSSConfig(host, orgID, name string, oss bool) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_project" "test_project" {
  name            = %[3]q
  organization_id = %[4]q
  oss             = %[5]t
}
`, host, fakeProjectToken, name, orgID, oss)
}

func projectImportSlug(s *terraform.State) (string, error) {
	slug, found := s.RootModule().Resources["circleci_project.test_project"].Primary.Attributes["slug"]
	if !found {
		return "", fmt.Errorf("attribute circleci_project.test_project.slug not found")
	}
	return slug, nil
}
