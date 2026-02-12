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

func TestAccProjectEnvironmentVariableResource(t *testing.T) {
	name := fmt.Sprintf("N%s", rand.Text())
	value := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectEnvironmentVariableResourceConfig(name, value, "circleci/8e4z1Akd74woxagxnvLT5q/CzMcAU8dvQo4FJhyj87QsA"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("value"),
						knownvalue.StringExact(value),
					),
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("project_slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q/CzMcAU8dvQo4FJhyj87QsA"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_project_environment_variable.test_env",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"value"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					projectSlug, found := s.RootModule().Resources["circleci_project_environment_variable.test_env"].Primary.Attributes["project_slug"]
					if !found {
						return "", errors.New("attribute circleci_project_environment_variable.test_env.project_slug not found")
					}
					envName, found := s.RootModule().Resources["circleci_project_environment_variable.test_env"].Primary.Attributes["name"]
					if !found {
						return "", errors.New("attribute circleci_project_environment_variable.test_env.name not found")
					}
					return fmt.Sprintf("%s/%s", projectSlug, envName), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccProjectEnvironmentVariableResourceUpdate(t *testing.T) {
	name := fmt.Sprintf("N%s", rand.Text())
	value := rand.Text()
	updatedValue := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectEnvironmentVariableResourceConfig(name, value, "circleci/8e4z1Akd74woxagxnvLT5q/CzMcAU8dvQo4FJhyj87QsA"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("value"),
						knownvalue.StringExact(value),
					),
				},
			},
			// Update (triggers replace) and Read testing
			{
				Config: testAccProjectEnvironmentVariableResourceConfig(name, updatedValue, "circleci/8e4z1Akd74woxagxnvLT5q/CzMcAU8dvQo4FJhyj87QsA"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"circleci_project_environment_variable.test_env",
						tfjsonpath.New("value"),
						knownvalue.StringExact(updatedValue),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectEnvironmentVariableResourceConfig(name, value, projectSlug string) string {
	return fmt.Sprintf(`
resource "circleci_project_environment_variable" "test_env" {
  project_slug = %[3]q
  name         = %[1]q
  value        = %[2]q
}
`, name, value, projectSlug)
}
