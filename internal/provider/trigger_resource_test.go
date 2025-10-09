// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTriggerResourceGithub(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceGithubAppConfig("61169e84-93ee-415d-8d65-ddf6dc0d2939", "fefb451c-9966-4b75-b555-d4d94d7116ef"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact("fefb451c-9966-4b75-b555-d4d94d7116ef"),
					),
				},
			},
		},
	})
}

func TestAccTriggerResourceWebhook(t *testing.T) {
	webhookTriggerName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceWebhookConfig(webhookTriggerName, "61169e84-93ee-415d-8d65-ddf6dc0d2939", "fefb451c-9966-4b75-b555-d4d94d7116ef"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_webhook",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_webhook",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact("fefb451c-9966-4b75-b555-d4d94d7116ef"),
					),
				},
			},
		},
	})
}

func testAccTriggerResourceGithubAppConfig(project_id, pipeline_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_github" {
  project_id 				= %[1]q
  pipeline_id 				= %[2]q
  event_source_provider = "github_app"
  event_source_repo_external_id = "952038793"
  event_preset = "all-pushes"
  checkout_ref = "some checkout ref github"
  config_ref = "some config ref github"
}
`, project_id, pipeline_id)
}

func testAccTriggerResourceWebhookConfig(event_name, project_id, pipeline_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_webhook" {
  event_name				= %[1]q
  project_id 				= %[2]q
  pipeline_id 				= %[3]q
  event_source_provider = "webhook"
  checkout_ref = "some checkout ref webhook"
  config_ref = "some config ref webhook"
}
`, event_name, project_id, pipeline_id)
}
