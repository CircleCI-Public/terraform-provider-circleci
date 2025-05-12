// Copyright (c) HashiCorp, Inc.
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

func TestAccContextEnvironmentVariableResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccContextEnvironmentVariableResourceConfig("one", "first_value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_context_environment_variable.test_env",
						tfjsonpath.New("context_id"),
						knownvalue.StringExact("e51158a2-f59c-4740-9eb4-d20609baa07e"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_environment_variable.test_env",
						tfjsonpath.New("name"),
						knownvalue.StringExact("one"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_environment_variable.test_env",
						tfjsonpath.New("value"),
						knownvalue.StringExact("first_value"),
					),
				},
			},
			// Update and Read testing
			{
				Config: testAccContextEnvironmentVariableResourceConfig("one", "second_value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_context_environment_variable.test_env",
						tfjsonpath.New("context_id"),
						knownvalue.StringExact("e51158a2-f59c-4740-9eb4-d20609baa07e"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_environment_variable.test_env",
						tfjsonpath.New("name"),
						knownvalue.StringExact("one"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_environment_variable.test_env",
						tfjsonpath.New("value"),
						knownvalue.StringExact("second_value"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccContextEnvironmentVariableResourceConfig(name, value string) string {
	return fmt.Sprintf(`
data "circleci_context" "test_context" {
  id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
}

resource "circleci_context_environment_variable" "test_env" {
  context_id = data.circleci_context.test_context.id
  name       = %[1]q
  value      = %[2]q
}
`, name, value)
}
