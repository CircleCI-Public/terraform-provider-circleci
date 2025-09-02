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

func TestAccContextDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testContextDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_context.test_context",
						tfjsonpath.New("id"),
						knownvalue.StringExact("e51158a2-f59c-4740-9eb4-d20609baa07e"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context.test_context",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Static Context"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context.test_context",
						tfjsonpath.New("created_at"),
						knownvalue.StringExact("2025-03-25T15:46:59.349Z"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context.test_context",
						tfjsonpath.New("restrictions"),

						knownvalue.ListExact(
							[]knownvalue.Check{
								knownvalue.MapPartial(
									map[string]knownvalue.Check{
										"id":         knownvalue.StringExact("94ab47c9-a9ee-4671-a176-49778ff578fd"),
										"project_id": knownvalue.StringExact("e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"),
										"name":       knownvalue.StringExact("test-project"),
										"type":       knownvalue.StringExact("project"),
										"value":      knownvalue.StringExact("e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"),
									},
								),
							},
						),
					),
				},
			},
		},
	})
}

const testContextDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2/"
}
  
data "circleci_context" "test_context" {
  id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
}
`
