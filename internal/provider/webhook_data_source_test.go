// Copyright (c) CircleCI
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

func TestAccWebhookDataSource(t *testing.T) {
	name := "webhook_test"
	projectId := "61169e84-93ee-415d-8d65-ddf6dc0d2939"
	webhookId := "06e947fc-b6f0-446c-b185-3699ea4e05e7"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create webhook resource first, then read with data source
			{
				Config: testAccWebhookDataSourceConfig(webhookId),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_webhook.test_webhook_data",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_webhook.test_webhook_data",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://xyz.circleci.com"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_webhook.test_webhook_data",
						tfjsonpath.New("verify_tls"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_webhook.test_webhook_data",
						tfjsonpath.New("scope_id"),
						knownvalue.StringExact(projectId),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_webhook.test_webhook_data",
						tfjsonpath.New("scope_type"),
						knownvalue.StringExact("project"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_webhook.test_webhook_data",
						tfjsonpath.New("events"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("workflow-completed"),
						}),
					),
				},
			},
		},
	})
}

func testAccWebhookDataSourceConfig(id string) string {
	return fmt.Sprintf(`
data "circleci_webhook" "test_webhook_data" {
  id = %[1]q
}
`, id)
}
