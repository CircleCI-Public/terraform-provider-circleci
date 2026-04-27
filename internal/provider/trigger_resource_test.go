// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
			// ImportState testing
			{
				ResourceName:            "circleci_trigger.test_trigger_github",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pipeline_id"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					triggerId, found := s.RootModule().Resources["circleci_trigger.test_trigger_github"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_github.id not found")
					}
					projectId, found := s.RootModule().Resources["circleci_trigger.test_trigger_github"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_github.project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectId, triggerId), nil
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
			// ImportState testing
			{
				ResourceName:            "circleci_trigger.test_trigger_webhook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"event_source_web_hook_url", "pipeline_id"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					triggerId, found := s.RootModule().Resources["circleci_trigger.test_trigger_webhook"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_webhook.id not found")
					}
					projectId, found := s.RootModule().Resources["circleci_trigger.test_trigger_webhook"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_webhook.project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectId, triggerId), nil
				},
			},
		},
	})
}

func TestAccTriggerResourceGithubServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceGithubServerConfig("20209578-aa1c-4b4c-9ca5-f6e38a47cf73", "9c7c4e85-5022-41d0-a6b0-705cfa856485"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github_server",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("20209578-aa1c-4b4c-9ca5-f6e38a47cf73"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github_server",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact("9c7c4e85-5022-41d0-a6b0-705cfa856485"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_trigger.test_trigger_github_server",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pipeline_id"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					triggerId, found := s.RootModule().Resources["circleci_trigger.test_trigger_github_server"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_github_server.id not found")
					}
					projectId, found := s.RootModule().Resources["circleci_trigger.test_trigger_github_server"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_github_server.project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectId, triggerId), nil
				},
			},
		},
	})
}

func TestAccTriggerResourceScheduled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceScheduledConfig(
					"e2e8ae23-57dc-4e95-bc67-633fdeb4ac33",
					"fefb451c-9966-4b75-b555-d4d94d7116ef",
					"0 * * * *",
					false,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact("fefb451c-9966-4b75-b555-d4d94d7116ef"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("event_source_provider"),
						knownvalue.StringExact("schedule"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("event_source_schedule_cron_expression"),
						knownvalue.StringExact("0 * * * *"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("disabled"),
						knownvalue.Bool(false),
					),
				},
			},
			// Update testing — change cron expression and disable the trigger
			{
				Config: testAccTriggerResourceScheduledConfig(
					"e2e8ae23-57dc-4e95-bc67-633fdeb4ac33",
					"fefb451c-9966-4b75-b555-d4d94d7116ef",
					"0 12 * * *",
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("event_source_schedule_cron_expression"),
						knownvalue.StringExact("0 12 * * *"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("disabled"),
						knownvalue.Bool(true),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_trigger.test_trigger_scheduled",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pipeline_id"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					triggerId, found := s.RootModule().Resources["circleci_trigger.test_trigger_scheduled"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_scheduled.id not found")
					}
					projectId, found := s.RootModule().Resources["circleci_trigger.test_trigger_scheduled"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute circleci_trigger.test_trigger_scheduled.project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectId, triggerId), nil
				},
			},
		},
	})
}

func testAccTriggerResourceScheduledConfig(project_id, pipeline_id, cron_expression string, disabled bool) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_scheduled" {
  project_id                              = %[1]q
  pipeline_id                             = %[2]q
  event_source_provider                   = "schedule"
  checkout_ref                            = "main"
  config_ref                              = "main"
  event_source_schedule_cron_expression   = %[3]q
  event_source_schedule_attribution_actor = "system"
  disabled                                = %[4]t
}
`, project_id, pipeline_id, cron_expression, disabled)
}

func testAccTriggerResourceGithubServerConfig(project_id, pipeline_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_github_server" {
  project_id                     = %[1]q
  pipeline_id                    = %[2]q
  event_source_provider          = "github_server"
  event_source_repo_external_id  = "2259"
  event_preset                   = "all-pushes"
  disabled                       = false
}
`, project_id, pipeline_id)
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
  disabled = false
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
  event_source_web_hook_sender = "web hook sender"
  disabled = false
}
`, event_name, project_id, pipeline_id)
}
