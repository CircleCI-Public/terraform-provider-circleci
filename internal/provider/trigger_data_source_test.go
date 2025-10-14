// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTriggerDataSource(t *testing.T) {
	dateRegex, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)
	if err != nil {
		t.Fatal("Could not create Date Regex for testing.")
	}
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
						knownvalue.StringExact("a7a10a1c-4818-464e-b233-50fd57e3c892"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("7d4d46da-49d1-4b3a-9a1b-3356ddfa67d6"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("checkout_ref"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_trigger.trigger_test",
						tfjsonpath.New("created_at"),
						knownvalue.StringRegexp(dateRegex),
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
						tfjsonpath.New("disabled"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

const testTriggerDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2"
}

data "circleci_trigger" "trigger_test" {
  id = "a7a10a1c-4818-464e-b233-50fd57e3c892"
  project_id = "7d4d46da-49d1-4b3a-9a1b-3356ddfa67d6"
}
`
