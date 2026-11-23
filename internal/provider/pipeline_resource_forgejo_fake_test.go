// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"

	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	forgejoPipelineAddress = "circleci_pipeline.build"
	forgejoRepoFullName    = "carta/service"
	forgejoRepoID          = "1234"
	forgejoOtherRepoName   = "carta/other"
	forgejoOtherRepoID     = "5678"
)

// newFakeForgejoOrg seeds a standalone organization in the fake, with two
// Forgejo repositories its connection will be able to see.
func newFakeForgejoOrg(t *testing.T) (fakeForgejo, string) {
	t.Helper()
	f := newFakeForgejo(t)
	org, err := f.fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: "carta"})
	assert.NilError(t, err)
	f.fake.AddForgejoRepo(forgejoRepoFullName, forgejoRepoID)
	f.fake.AddForgejoRepo(forgejoOtherRepoName, forgejoOtherRepoID)
	return f, org.ID.String()
}

// testAccForgejoOrgConfig configures what every Forgejo pipeline needs: a
// project in orgID and the org's Forgejo connection.
func testAccForgejoOrgConfig(host, orgID string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_project" "service" {
  name            = "service"
  organization_id = %[3]q
}

resource "circleci_forgejo_connection" "carta" {
  organization_id     = %[3]q
  url                 = "https://forgejo.example.com"
  owner               = "carta"
  connection_audience = %[4]q
  checkout_audience   = %[5]q
}
`, host, fakeForgejoToken, orgID, fakeForgejoConnectionAudience, fakeForgejoCheckoutAudience)
}

func testAccForgejoPipelineConfig(host, orgID, repo, description string) string {
	return testAccForgejoOrgConfig(host, orgID) + fmt.Sprintf(`
resource "circleci_pipeline" "build" {
  project_id                     = circleci_project.service.id
  name                           = "build"
  description                    = %[2]q
  config_source_provider         = "forgejo"
  config_source_repo_full_name   = %[1]q
  config_source_file_path        = ".circleci/config.yml"
  checkout_source_provider       = "forgejo"
  checkout_source_repo_full_name = %[1]q

  depends_on = [circleci_forgejo_connection.carta]
}
`, repo, description)
}

func TestAccPipelineResourceForgejo(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccForgejoPipelineConfig(f.host, orgID, forgejoRepoFullName, "Builds the service"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoPipelineAddress, tfjsonpath.New("config_source_repo_external_id"), knownvalue.StringExact(forgejoRepoID)),
					statecheck.ExpectKnownValue(forgejoPipelineAddress, tfjsonpath.New("checkout_source_repo_external_id"), knownvalue.StringExact(forgejoRepoID)),
					statecheck.ExpectKnownValue(forgejoPipelineAddress, tfjsonpath.New("checkout_source_repo_full_name"), knownvalue.StringExact(forgejoRepoFullName)),
				},
			},
			{
				Config: testAccForgejoPipelineConfig(f.host, orgID, forgejoRepoFullName, "Builds and tests the service"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoPipelineAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(forgejoPipelineAddress, tfjsonpath.New("checkout_source_repo_external_id"), knownvalue.StringExact(forgejoRepoID)),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoPipelineAddress, tfjsonpath.New("description"), knownvalue.StringExact("Builds and tests the service")),
				},
			},
			{
				Config: testAccForgejoPipelineConfig(f.host, orgID, forgejoOtherRepoName, "Builds and tests the service"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoPipelineAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoPipelineAddress, tfjsonpath.New("checkout_source_repo_external_id"), knownvalue.StringExact(forgejoOtherRepoID)),
				},
			},
			{
				ResourceName:      forgejoPipelineAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[forgejoPipelineAddress]
					return rs.Primary.Attributes["project_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccPipelineResourceForgejoConnectionDeleted(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)
	config := testAccForgejoPipelineConfig(f.host, orgID, forgejoRepoFullName, "Builds the service")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					id := f.fake.ConnectionIDForOrg(orgID)
					assert.Assert(t, id != "")
					err := f.connections().Delete(context.TODO(), id)
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("circleci_forgejo_connection.carta", plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(forgejoPipelineAddress, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccPipelineResourceForgejoInvalid(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)
	base := testAccForgejoOrgConfig(f.host, orgID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: base + `
resource "circleci_pipeline" "build" {
  project_id               = circleci_project.service.id
  name                     = "build"
  description              = ""
  config_source_provider   = "forgejo"
  config_source_file_path  = ".circleci/config.yml"
  checkout_source_provider = "forgejo"
  checkout_source_repo_full_name = "carta/service"
}
`,
				ExpectError: regexp.MustCompile(`A forgejo pipeline needs config_source_repo_external_id or`),
			},
			{
				Config: base + `
resource "circleci_pipeline" "build" {
  project_id                       = circleci_project.service.id
  name                             = "build"
  description                      = ""
  config_source_provider           = "forgejo"
  config_source_repo_full_name     = "carta/service"
  config_source_file_path          = ".circleci/config.yml"
  checkout_source_provider         = "github_app"
  checkout_source_repo_external_id = "1"
}
`,
				ExpectError: regexp.MustCompile(`Mismatched providers`),
			},
			{
				Config: base + `
resource "circleci_pipeline" "build" {
  project_id                       = circleci_project.service.id
  name                             = "build"
  description                      = ""
  config_source_provider           = "github_app"
  config_source_repo_external_id   = "1"
  config_source_repo_full_name     = "org/repo"
  config_source_file_path          = ".circleci/config.yml"
  checkout_source_provider         = "github_app"
  checkout_source_repo_external_id = "1"
}
`,
				ExpectError: regexp.MustCompile(`can only be set for forgejo`),
			},
		},
	})
}
