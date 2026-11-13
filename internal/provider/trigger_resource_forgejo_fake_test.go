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
)

const forgejoTriggerAddress = "circleci_trigger.prs"

func testAccForgejoTriggerConfig(host, orgID, trigger string) string {
	return testAccForgejoPipelineConfig(host, orgID, forgejoRepoFullName, "Builds the service") + trigger
}

func forgejoTrigger(preset string, disabled bool) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "prs" {
  project_id                  = circleci_project.service.id
  pipeline_id                 = circleci_pipeline.build.id
  event_source_provider       = "forgejo"
  event_source_repo_external_id = circleci_pipeline.build.checkout_source_repo_external_id
  event_preset                  = %q
  disabled                      = %t
}
`, preset, disabled)
}

func TestAccTriggerResourceForgejo(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccForgejoTriggerConfig(f.host, orgID, forgejoTrigger("only-open-prs", false)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("event_source_repo_external_id"), knownvalue.StringExact(forgejoRepoID)),
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("event_preset"), knownvalue.StringExact("only-open-prs")),
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("checkout_ref"), knownvalue.Null()),
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("config_ref"), knownvalue.Null()),
				},
			},
			{
				Config: testAccForgejoTriggerConfig(f.host, orgID, forgejoTrigger("only-merged-prs", true)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoTriggerAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("event_preset"), knownvalue.StringExact("only-merged-prs")),
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("disabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(forgejoTriggerAddress, tfjsonpath.New("event_source_repo_external_id"), knownvalue.StringExact(forgejoRepoID)),
				},
			},
			{
				ResourceName:      forgejoTriggerAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[forgejoTriggerAddress]
					return rs.Primary.Attributes["project_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccTriggerResourceForgejoConnectionDeleted(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)
	config := testAccForgejoTriggerConfig(f.host, orgID, forgejoTrigger("all-pushes", false))

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
						plancheck.ExpectResourceAction(forgejoPipelineAddress, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(forgejoTriggerAddress, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccTriggerResourceForgejoInvalid(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccForgejoTriggerConfig(f.host, orgID, `
resource "circleci_trigger" "prs" {
  project_id            = circleci_project.service.id
  pipeline_id           = circleci_pipeline.build.id
  event_source_provider = "forgejo"
  event_preset          = "all-pushes"
}
`),
				ExpectError: regexp.MustCompile(`A forgejo trigger needs event_source_repo_external_id`),
			},
			{
				Config:      testAccForgejoTriggerConfig(f.host, orgID, forgejoTrigger("only-labeled-prs", false)),
				ExpectError: regexp.MustCompile(`forgejo provider has an unexpected event_preset`),
			},
		},
	})
}

// A trigger that reads its repository ID from its pipeline must be updated,
// not replaced, when the pipeline is updated in the same apply.
func TestAccTriggerResourceForgejoPipelineUpdated(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccForgejoTriggerConfig(f.host, orgID, forgejoTrigger("all-pushes", false))},
			{
				Config: testAccForgejoPipelineConfig(f.host, orgID, forgejoRepoFullName, "Builds and tests the service") +
					forgejoTrigger("only-merged-prs", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoPipelineAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(forgejoTriggerAddress, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}
