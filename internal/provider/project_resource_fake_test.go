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

func TestAccProjectResourceOSSFalseWhenNotOpenSource(t *testing.T) {
	fc := fakecircle.New(fakeProjectToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	org, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "closed org false",
	})
	assert.NilError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceOSSConfig(srv.URL+"/api/v2", org.ID.String(), "closed-source-false", false),
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
		},
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
