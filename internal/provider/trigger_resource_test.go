// Copyright (c) CircleCI and HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0



package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTriggerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceConfig("name2", "61169e84-93ee-415d-8d65-ddf6dc0d2939", "fefb451c-9966-4b75-b555-d4d94d7116ef"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger",
						tfjsonpath.New("name"),
						knownvalue.StringExact("name2"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact("fefb451c-9966-4b75-b555-d4d94d7116ef"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTriggerResourceConfig(name, project_id, pipeline_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger" {
  project_id 				= %[2]q
  pipeline_id 				= %[3]q
  name 						= %[1]q
  description = "some description"
  event_source_provider = "github_app"
  event_source_repo_external_id = "952038793"
  event_preset = "all-pushes"
  checkout_ref = "some checkout ref"
  config_ref = "some config ref2"
}
`, name, project_id, pipeline_id)
}
