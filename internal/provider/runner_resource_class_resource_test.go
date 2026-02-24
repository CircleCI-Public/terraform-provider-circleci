// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRunnerResourceClassResource(t *testing.T) {
	resourceClass := "cci-terraform-test/acc-test-runner"
	description := "Acceptance test runner resource class"
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRunnerResourceClassConfig(resourceClass, description, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact(resourceClass),
					),
					statecheck.ExpectKnownValue(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(description),
					),
					statecheck.ExpectKnownValue(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_runner_resource_class.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"force_delete"},
				ImportStateId:           resourceClass,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccRunnerResourceClassForceDelete(t *testing.T) {
	resourceClass := "cci-terraform-test/acc-test-runner-force"
	description := "Acceptance test runner resource class with force delete"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with force_delete = true
			{
				Config: testAccRunnerResourceClassConfig(resourceClass, description, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact(resourceClass),
					),
					statecheck.ExpectKnownValue(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(description),
					),
					statecheck.ExpectKnownValue(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("force_delete"),
						knownvalue.Bool(true),
					),
				},
			},
			// Delete testing automatically occurs in TestCase (exercises force-delete path)
		},
	})
}

func testAccRunnerResourceClassConfig(resourceClass, description string, forceDelete bool) string {
	return fmt.Sprintf(`
resource "circleci_runner_resource_class" "test" {
  resource_class = %[1]q
  description    = %[2]q
  force_delete   = %[3]t
}
`, resourceClass, description, forceDelete)
}
