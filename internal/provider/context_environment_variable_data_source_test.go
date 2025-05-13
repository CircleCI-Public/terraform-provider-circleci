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

func TestAccContextEnvironmentVariableDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testContextEnvironmentVariableDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_context_environment_variable.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TEST1"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context_environment_variable.test",
						tfjsonpath.New("context_id"),
						knownvalue.StringExact("e51158a2-f59c-4740-9eb4-d20609baa07e"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context_environment_variable.test",
						tfjsonpath.New("created_at"),
						knownvalue.StringExact("2025-03-27T14:35:31.435Z"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context_environment_variable.test",
						tfjsonpath.New("updated_at"),
						knownvalue.StringExact("2025-03-27T14:35:31.435Z"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context_environment_variable.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TEST1"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_context_environment_variable.test",
						tfjsonpath.New("context_id"),
						knownvalue.StringExact("e51158a2-f59c-4740-9eb4-d20609baa07e"),
					),
				},
			},
		},
	})
}

var testContextEnvironmentVariableDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2/"
}

data "circleci_context_environment_variable" "test" {
  name = "TEST1"
  context_id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
}
`
