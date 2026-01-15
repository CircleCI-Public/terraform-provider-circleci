// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccWebhookResource(t *testing.T) {
	dateRegex, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$`)
	if err != nil {
		t.Fatal("Could not create Date Regex for testing.")
	}
	randName := rand.Text()
	projectId := "61169e84-93ee-415d-8d65-ddf6dc0d2939"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWebhookResourceConfig(randName, projectId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("name"),
						knownvalue.StringExact(randName),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://example.com/webhook"),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("verify_tls"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("scope_id"),
						knownvalue.StringExact(projectId),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("scope_type"),
						knownvalue.StringExact("project"),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("events"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("workflow-completed"),
						}),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("created_at"),
						knownvalue.StringRegexp(dateRegex),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_webhook.test_webhook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"signing_secret", "verify_tls"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					webhookId, found := s.RootModule().Resources["circleci_webhook.test_webhook"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_webhook.test_webhook.id not found")
					}
					scopeId, found := s.RootModule().Resources["circleci_webhook.test_webhook"].Primary.Attributes["scope_id"]
					if !found {
						return "", errors.New("attribute circleci_webhook.test_webhook.scope_id not found")
					}
					return fmt.Sprintf("%s/%s", scopeId, webhookId), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccWebhookResourceUpdate(t *testing.T) {
	randName := rand.Text()
	updatedName := rand.Text()
	projectId := "61169e84-93ee-415d-8d65-ddf6dc0d2939"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccWebhookResourceConfig(randName, projectId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("name"),
						knownvalue.StringExact(randName),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("events"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("workflow-completed"),
						}),
					),
				},
			},
			// Update
			{
				Config: testAccWebhookResourceConfigUpdated(updatedName, projectId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("name"),
						knownvalue.StringExact(updatedName),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://example.com/webhook-updated"),
					),
					statecheck.ExpectKnownValue(
						"circleci_webhook.test_webhook",
						tfjsonpath.New("events"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("workflow-completed"),
							knownvalue.StringExact("job-completed"),
						}),
					),
				},
			},
		},
	})
}

func testAccWebhookResourceConfig(name, scopeId string) string {
	return fmt.Sprintf(`
resource "circleci_webhook" "test_webhook" {
  name           = %[1]q
  url            = "https://example.com/webhook"
  verify_tls     = true
  signing_secret = "secret"
  scope_id       = %[2]q
  scope_type     = "project"
  events         = ["workflow-completed"]
}
`, name, scopeId)
}

func testAccWebhookResourceConfigUpdated(name, scopeId string) string {
	return fmt.Sprintf(`
resource "circleci_webhook" "test_webhook" {
  name           = %[1]q
  url            = "https://example.com/webhook-updated"
  verify_tls     = true
  signing_secret = "secret"
  scope_id       = %[2]q
  scope_type     = "project"
  events         = ["workflow-completed", "job-completed"]
}
`, name, scopeId)
}
