// Copyright (c) HashiCorp, Inc.
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

func TestAccProjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig(
					"name",
					"circleci",
					"cci-terraform-test",
					"circleci/8e4z1Akd74woxagxnvLT5q",
					"3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact("name"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("project_provider"),
						knownvalue.StringExact("circleci"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact("3ddcf1d1-7f5f-4139-8cef-71ad0921a968"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfig(name, project_provider, organization_name, organization_slug, organization_id string) string {
	return fmt.Sprintf(`
resource "circleci_project" "test_project" {
  name 				= %[1]q
  project_provider 	= %[2]q
  organization_name = %[3]q
  organization_slug = %[4]q
  organization_id 	= %[5]q
}
`, name, project_provider, organization_name, organization_slug, organization_id)
}
