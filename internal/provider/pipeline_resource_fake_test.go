// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	fakePipelineToken          = "fake-pipeline-token"
	fakePipelineProjectID      = "00000000-0000-0000-0000-000000000011"
	fakePipelineRepoExternalID = "1028689591"
	fakePipelineOtherRepoID    = "1028689592"
	fakePipelineName           = "fake pipeline"
	fakePipelineConfigPath     = ".circleci/config.yml"
)

func testAccPipelineResourceFakeConfig(host, description, checkoutRepoExternalID string, withTrigger bool) string {
	config := fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_pipeline" "test_pipeline" {
  project_id                       = %[3]q
  name                             = %[4]q
  description                      = %[5]q
  config_source_provider           = "github_app"
  config_source_file_path          = %[6]q
  config_source_repo_external_id   = %[7]q
  checkout_source_provider         = "github_app"
  checkout_source_repo_external_id = %[8]q
}
`, host, fakePipelineToken, fakePipelineProjectID, fakePipelineName, description,
		fakePipelineConfigPath, fakePipelineRepoExternalID, checkoutRepoExternalID)

	if withTrigger {
		config += fmt.Sprintf(`
resource "circleci_trigger" "test_trigger" {
  project_id                    = %[1]q
  pipeline_id                   = circleci_pipeline.test_pipeline.id
  event_source_provider         = "github_app"
  event_source_repo_external_id = %[2]q
  event_preset                  = "all-pushes"
  disabled                      = false
}
`, fakePipelineProjectID, fakePipelineRepoExternalID)
	}

	return config
}

// Changing only the description of a pipeline definition that has a trigger
// must update it in place. The API rejects any checkout_source in the update
// while triggers exist, so the request must not carry one.
func TestAccPipelineResourceUpdateDescriptionWithTrigger(t *testing.T) {
	fc := fakecircle.New(fakePipelineToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)
	host := srv.URL + "/api/v2"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineResourceFakeConfig(host, "original description", fakePipelineRepoExternalID, true),
			},
			{
				Config: testAccPipelineResourceFakeConfig(host, "updated description", fakePipelineRepoExternalID, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"circleci_pipeline.test_pipeline",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("description"),
						knownvalue.StringExact("updated description"),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("checkout_source_repo_external_id"),
						knownvalue.StringExact(fakePipelineRepoExternalID),
					),
				},
			},
		},
	})

	updates := fc.PipelineDefinitionUpdates()
	assert.Assert(t, cmp.Len(updates, 1))
	assert.Check(t, cmp.DeepEqual(updates[0], map[string]any{
		"name":        fakePipelineName,
		"description": "updated description",
		"config_source": map[string]any{
			"file_path": fakePipelineConfigPath,
		},
	}))
}

// Changing the checkout source must still send it.
func TestAccPipelineResourceUpdateCheckoutSource(t *testing.T) {
	fc := fakecircle.New(fakePipelineToken)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)
	host := srv.URL + "/api/v2"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineResourceFakeConfig(host, "description", fakePipelineRepoExternalID, false),
			},
			{
				Config: testAccPipelineResourceFakeConfig(host, "description", fakePipelineOtherRepoID, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"circleci_pipeline.test_pipeline",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("checkout_source_repo_external_id"),
						knownvalue.StringExact(fakePipelineOtherRepoID),
					),
				},
			},
		},
	})

	updates := fc.PipelineDefinitionUpdates()
	assert.Assert(t, cmp.Len(updates, 1))
	assert.Check(t, cmp.DeepEqual(updates[0], map[string]any{
		"name":        fakePipelineName,
		"description": "description",
		"config_source": map[string]any{
			"file_path": fakePipelineConfigPath,
		},
		"checkout_source": map[string]any{
			"provider": "github_app",
			"repo": map[string]any{
				"external_id": fakePipelineOtherRepoID,
			},
		},
	}))
}
