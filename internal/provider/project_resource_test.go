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

func TestAccProjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig("name", "project_provider", "organization_slug_part"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact("name"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("project_provider"),
						knownvalue.StringExact("project_provider"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_slug_part"),
						knownvalue.StringExact("organization_slug_part"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfig(name, project_provider, organization_slug_part string) string {
	return fmt.Sprintf(`
resource "circleci_project" "test_project" {
  name 						= %[1]q
  project_provider 			= %[2]q
  organization_slug_part 	= %[3]q
}
`, name, project_provider, organization_slug_part)
}
