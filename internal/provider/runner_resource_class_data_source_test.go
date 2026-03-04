// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRunnerResourceClassDataSource(t *testing.T) {
	organizationId := "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
	resourceClass := "cci-terraform-test/acc-test-runner-ds"
	description := "Acceptance test runner resource class data source"
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a resource class then read it back via the data source.
			// Verifies all three attributes and that the data source id matches
			// the resource id.
			{
				Config: testAccRunnerResourceClassDataSourceConfig(organizationId, resourceClass, description),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_runner_resource_class.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact(resourceClass),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_runner_resource_class.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(description),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_runner_resource_class.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					// Data source id must equal the resource id.
					statecheck.CompareValuePairs(
						"circleci_runner_resource_class.test",
						tfjsonpath.New("id"),
						"data.circleci_runner_resource_class.test",
						tfjsonpath.New("id"),
						compare.ValuesSame(),
					),
				},
			},
		},
	})
}

func TestAccRunnerResourceClassDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRunnerResourceClassDataSourceOnlyConfig("cci-terraform-test/does-not-exist-acc"),
				ExpectError: regexp.MustCompile(`Runner resource class not found`),
			},
		},
	})
}

func TestAccRunnerResourceClassDataSourceInvalidFormat(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRunnerResourceClassDataSourceOnlyConfig("noslash"),
				ExpectError: regexp.MustCompile(`Invalid resource_class format`),
			},
		},
	})
}

func testAccRunnerResourceClassDataSourceConfig(organizationId, resourceClass, description string) string {
	return fmt.Sprintf(`
resource "circleci_runner_resource_class" "test" {
  organization_id = %[1]q
  resource_class = %[2]q
  description    = %[3]q
}

data "circleci_runner_resource_class" "test" {
  organization_id = circleci_runner_resource_class.test.organization_id
  resource_class = circleci_runner_resource_class.test.resource_class
}
`, organizationId, resourceClass, description)
}

func testAccRunnerResourceClassDataSourceOnlyConfig(resourceClass string) string {
	return fmt.Sprintf(`
data "circleci_runner_resource_class" "test" {
  resource_class = %[1]q
}
`, resourceClass)
}
