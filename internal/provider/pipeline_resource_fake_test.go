// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gotest.tools/v3/assert"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/pipeline"
	"terraform-provider-circleci/internal/circleci/trigger"
)

const githubPipelineAddress = "circleci_pipeline.github"

func testAccPipelineResourceGithubAppConfig(host string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_pipeline" "github" {
  project_id                       = %[3]q
  name                             = "build"
  description                      = "Builds the service"
  config_source_provider           = "github_app"
  config_source_file_path          = ".circleci/config.yml"
  config_source_repo_external_id   = %[4]q
  checkout_source_provider         = "github_app"
  checkout_source_repo_external_id = %[4]q
}

resource "circleci_trigger" "github" {
  project_id                    = %[3]q
  pipeline_id                   = circleci_pipeline.github.id
  event_source_provider         = "github_app"
  event_source_repo_external_id = %[4]q
  event_preset                  = "all-pushes"
}
`, host, fakeTriggerToken, fakeTriggerProjectID, fakeTriggerRepoExternalID)
}

// A pipeline and its trigger deleted outside Terraform must be planned for
// creation rather than failing the refresh. CircleCI refuses to delete a
// pipeline that still has triggers, so the trigger goes first.
func TestAccPipelineResourceDeletedOutOfBand(t *testing.T) {
	host := newFakeTriggerHost(t)
	c := client.NewClient(host, fakeTriggerToken, "terraform-provider-circleci/test")
	pipelines := pipeline.NewPipelineService(c)
	triggers := trigger.NewTriggerService(c)
	config := testAccPipelineResourceGithubAppConfig(host)

	var pipelineID, triggerID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					pipelineID = s.RootModule().Resources[githubPipelineAddress].Primary.ID
					triggerID = s.RootModule().Resources["circleci_trigger.github"].Primary.ID
					return nil
				},
			},
			{
				PreConfig: func() {
					err := triggers.Delete(context.TODO(), fakeTriggerProjectID, triggerID)
					assert.NilError(t, err)
					err = pipelines.Delete(context.TODO(), fakeTriggerProjectID, pipelineID)
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(githubPipelineAddress, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("circleci_trigger.github", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

// Replacing a pipeline must replace its triggers too. CircleCI refuses to
// delete a pipeline that still has triggers, and a trigger cannot move to
// another pipeline.
func TestAccPipelineResourceReplacedWithTrigger(t *testing.T) {
	host := newFakeTriggerHost(t)
	config := testAccPipelineResourceGithubAppConfig(host)
	renamed := strings.Replace(config, `name                             = "build"`, `name                             = "build-and-test"`, 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config: renamed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(githubPipelineAddress, plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectResourceAction("circleci_trigger.github", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}

// Updating a pipeline in place keeps its ID, so its triggers, which are
// replaced when their pipeline_id changes, must be left alone.
func TestAccPipelineResourceUpdatedKeepsTrigger(t *testing.T) {
	host := newFakeTriggerHost(t)
	config := testAccPipelineResourceGithubAppConfig(host)
	described := strings.Replace(config, `description                      = "Builds the service"`, `description                      = "Builds and tests the service"`, 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config: described,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(githubPipelineAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("circleci_trigger.github", plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}
