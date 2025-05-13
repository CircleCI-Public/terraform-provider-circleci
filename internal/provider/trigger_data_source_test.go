// Copyright (c) HashiCorp, Inc.
// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTriggerDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testTriggerDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("88df1577-6df5-44e0-a5b0-acecffa78590"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("checkout_ref"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("created_at"),
						knownvalue.StringExact("2025-04-09T16:11:20.238118Z"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("event_preset"),
						knownvalue.StringExact("all-pushes"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("event_source_provider"),
						knownvalue.StringExact("github_app"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("event_source_repository_external_id"),
						knownvalue.StringExact("952038793"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("event_source_repository_name"),
						knownvalue.StringExact("cci-terraform-test/test-repo"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("event_source_webhook_url"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("github_test_repo_trigger"),
					),
				},
			},
		},
	})
}

const testTriggerDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2/"
}

data "circleci_trigger" "trigger_test" {
  id = "88df1577-6df5-44e0-a5b0-acecffa78590"
  project_id = "e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"
}
`
