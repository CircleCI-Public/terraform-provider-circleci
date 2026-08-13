// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/trigger"
)

// testAccCleanupRepoTrigger deletes any trigger already attached to a pipeline
// definition for the given event source provider and repository.
//
// The repo-backed trigger tests target fixed project and pipeline-definition
// IDs, so a run that dies before its destroy step leaves a trigger behind, and
// the leftover then has to be removed by hand before the test can create its
// own again. Matching on provider and repository rather than deleting every
// trigger keeps this from disturbing any fixture that shares the pipeline
// definition.
func testAccCleanupRepoTrigger(t *testing.T, projectID, pipelineID, provider, repoExternalID string) {
	t.Helper()

	host := os.Getenv("CIRCLE_HOST")
	if host == "" {
		host = "https://circleci.com/api/v2"
	}

	svc := trigger.NewTriggerService(
		client.NewClient(host, os.Getenv("CIRCLE_TOKEN"), "terraform-provider-circleci/test"),
	)

	ctx := context.Background()
	existing, err := svc.List(ctx, projectID, pipelineID)
	if err != nil {
		t.Fatalf("listing triggers on pipeline definition %s: %v", pipelineID, err)
	}

	for _, tr := range existing {
		if tr.EventSource.Provider != provider || tr.EventSource.Repo.ExternalId != repoExternalID {
			continue
		}
		if err := svc.Delete(ctx, projectID, tr.ID); err != nil {
			t.Fatalf("deleting leftover %s trigger %s: %v", provider, tr.ID, err)
		}
		t.Logf("deleted leftover %s trigger %s (created %s)", provider, tr.ID, tr.CreatedAt)
	}
}

func TestAccTriggerResourceGithub(t *testing.T) {
	const (
		projectID      = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
		pipelineID     = "fefb451c-9966-4b75-b555-d4d94d7116ef"
		repoExternalID = "952038793"
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCleanupRepoTrigger(t, projectID, pipelineID, "github_app", repoExternalID)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceGithubAppConfig(projectID, pipelineID, repoExternalID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(projectID),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact(pipelineID),
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
				Config: testAccTriggerResourceWebhookConfig(webhookTriggerName, "61169e84-93ee-415d-8d65-ddf6dc0d2939", "fefb451c-9966-4b75-b555-d4d94d7116ef", nil),
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
	const (
		projectID      = "20209578-aa1c-4b4c-9ca5-f6e38a47cf73"
		pipelineID     = "9c7c4e85-5022-41d0-a6b0-705cfa856485"
		repoExternalID = "2259"
	)

	// Quarantined, not a provider bug: creating a trigger on this project hangs
	// for 20s in soc-integrations and returns 504, which reaches us as a 500.
	// 64/64 requests fail, while eight other projects create triggers in
	// 250-680ms on the same route. Broken since 11 Aug 2026 ~20:00 UTC with no
	// corresponding deploy. Re-enable once the API is fixed.
	// https://circleci.atlassian.net/wiki/spaces/~712020a3402082e9a44c0bb1238ea0280d1305/pages/9001500676
	t.Skip("blocked on soc-integrations 504 creating github_server triggers for project " + projectID)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCleanupRepoTrigger(t, projectID, pipelineID, "github_server", repoExternalID)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceGithubServerConfig(projectID, pipelineID, repoExternalID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github_server",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact(projectID),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github_server",
						tfjsonpath.New("pipeline_id"),
						knownvalue.StringExact(pipelineID),
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
	pipelineName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTriggerResourceScheduledConfig(
					pipelineName,
					"0 * * * *",
					false,
					map[string]any{"run_nightly_foo": true, "retries": 3, "branch": "main"},
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
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
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("parameters"),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"run_nightly_foo": knownvalue.Bool(true),
							"retries":         knownvalue.NumberExact(big.NewFloat(3)),
							"branch":          knownvalue.StringExact("main"),
						}),
					),
				},
			},
			// Update testing — change cron expression, disable the trigger, and flip one parameter
			{
				Config: testAccTriggerResourceScheduledConfig(
					pipelineName,
					"0 12 * * *",
					true,
					map[string]any{"run_nightly_foo": false, "retries": 3, "branch": "main"},
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
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("parameters"),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"run_nightly_foo": knownvalue.Bool(false),
							"retries":         knownvalue.NumberExact(big.NewFloat(3)),
							"branch":          knownvalue.StringExact("main"),
						}),
					),
				},
			},
			// Removing the parameters block should clear them on the API
			{
				Config: testAccTriggerResourceScheduledConfig(
					pipelineName,
					"0 12 * * *",
					true,
					nil,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("parameters"),
						knownvalue.Null(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_trigger.test_trigger_scheduled",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pipeline_id", "event_source_schedule_attribution_actor"},
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

func TestAccTriggerResourceScheduledNoParameters(t *testing.T) {
	pipelineName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerResourceScheduledConfig(
					pipelineName,
					"0 * * * *",
					false,
					nil,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_scheduled",
						tfjsonpath.New("parameters"),
						knownvalue.Null(),
					),
				},
			},
			{
				ResourceName:            "circleci_trigger.test_trigger_scheduled",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pipeline_id", "event_source_schedule_attribution_actor"},
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

func TestAccTriggerResourceWebhookRejectsParameters(t *testing.T) {
	webhookTriggerName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerResourceWebhookConfig(
					webhookTriggerName,
					"61169e84-93ee-415d-8d65-ddf6dc0d2939",
					"fefb451c-9966-4b75-b555-d4d94d7116ef",
					map[string]any{"foo": "bar"},
				),
				ExpectError: regexp.MustCompile("does not support parameters"),
			},
		},
	})
}

func testAccTriggerResourceScheduledConfig(pipeline_name, cron_expression string, disabled bool, parameters map[string]any) string {
	return fmt.Sprintf(`
resource "circleci_pipeline" "test_pipeline_scheduled" {
  project_id                       = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
  name                             = %[1]q
  description                      = "pipeline for scheduled trigger acceptance test"
  config_source_provider           = "github_app"
  config_source_file_path          = ".circleci/config.yml"
  config_source_repo_external_id   = "952038793"
  checkout_source_provider         = "github_app"
  checkout_source_repo_external_id = "952038793"
}

resource "circleci_trigger" "test_trigger_scheduled" {
  project_id                              = circleci_pipeline.test_pipeline_scheduled.project_id
  pipeline_id                             = circleci_pipeline.test_pipeline_scheduled.id
  event_source_provider                   = "schedule"
  event_name                              = "scheduled_pipeline"
  checkout_ref                            = "main"
  config_ref                              = "main"
  event_source_schedule_cron_expression   = %[2]q
  event_source_schedule_attribution_actor = "system"
  disabled                                = %[3]t
%[4]s}
`, pipeline_name, cron_expression, disabled, renderParametersHCL(parameters))
}

func renderParametersHCL(parameters map[string]any) string {
	if len(parameters) == 0 {
		return ""
	}
	keys := make([]string, 0, len(parameters))
	for k := range parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("  parameters = {\n")
	for _, k := range keys {
		switch v := parameters[k].(type) {
		case string:
			fmt.Fprintf(&b, "    %q = %q\n", k, v)
		default:
			// bool and numeric values render unquoted so Terraform preserves their type.
			fmt.Fprintf(&b, "    %q = %v\n", k, v)
		}
	}
	b.WriteString("  }\n")
	return b.String()
}

func testAccTriggerResourceGithubServerConfig(project_id, pipeline_id, repo_external_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_github_server" {
  project_id                     = %[1]q
  pipeline_id                    = %[2]q
  event_source_provider          = "github_server"
  event_source_repo_external_id  = %[3]q
  event_preset                   = "all-pushes"
  disabled                       = false
}
`, project_id, pipeline_id, repo_external_id)
}

func testAccTriggerResourceGithubAppConfig(project_id, pipeline_id, repo_external_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_github" {
  project_id 				= %[1]q
  pipeline_id 				= %[2]q
  event_source_provider = "github_app"
  event_source_repo_external_id = %[3]q
  event_preset = "all-pushes"
  checkout_ref = "some checkout ref github"
  config_ref = "some config ref github"
  disabled = false
}
`, project_id, pipeline_id, repo_external_id)
}

func testAccTriggerResourceGithubAppConfigNoRepoExternalId(project_id, pipeline_id string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_github" {
  project_id            = %[1]q
  pipeline_id           = %[2]q
  event_source_provider = "github_app"
  event_preset          = "all-pushes"
  checkout_ref          = "some checkout ref github"
  config_ref            = "some config ref github"
  disabled              = false
}
`, project_id, pipeline_id)
}

func testAccTriggerResourceWebhookConfig(event_name, project_id, pipeline_id string, parameters map[string]any) string {
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
%[4]s}
`, event_name, project_id, pipeline_id, renderParametersHCL(parameters))
}

func TestAccTriggerResourceUpdateRemovesRepoExternalId(t *testing.T) {
	const (
		projectID      = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
		pipelineID     = "fefb451c-9966-4b75-b555-d4d94d7116ef"
		repoExternalID = "952038793"
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCleanupRepoTrigger(t, projectID, pipelineID, "github_app", repoExternalID)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTriggerResourceGithubAppConfig(projectID, pipelineID, repoExternalID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_github",
						tfjsonpath.New("event_source_repo_external_id"),
						knownvalue.StringExact(repoExternalID),
					),
				},
			},
			{
				Config:      testAccTriggerResourceGithubAppConfigNoRepoExternalId(projectID, pipelineID),
				ExpectError: regexp.MustCompile(`requires[\s]+event_source_repo_external_id`),
			},
		},
	})
}

func TestAccTriggerResourceMissingRepoExternalId(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "circleci_trigger" "test_missing_repo_id" {
  project_id            = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
  pipeline_id           = "fefb451c-9966-4b75-b555-d4d94d7116ef"
  event_source_provider = "github_app"
  event_preset          = "all-pushes"
}
`,
				// [\s]+ tolerates the newline Terraform CLI inserts when word-wrapping diagnostics.
				ExpectError: regexp.MustCompile(`requires[\s]+event_source_repo_external_id`),
			},
		},
	})
}
