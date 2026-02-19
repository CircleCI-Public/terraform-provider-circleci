// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOrganizationCircleCiResource(t *testing.T) {
	organizationName := rand.Text()
	slugRegex := regexp.MustCompile(`^circleci/[a-zA-Z0-9._-]+$`)
	idRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOrganizationResourceConfig(
					organizationName,
					"circleci",
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_organization.test_organization",
						tfjsonpath.New("name"),
						knownvalue.StringExact(organizationName),
					),
					statecheck.ExpectKnownValue(
						"circleci_organization.test_organization",
						tfjsonpath.New("slug"),
						knownvalue.StringRegexp(slugRegex),
					),
					statecheck.ExpectKnownValue(
						"circleci_organization.test_organization",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(idRegex),
					),
					statecheck.ExpectKnownValue(
						"circleci_organization.test_organization",
						tfjsonpath.New("vcs_type"),
						knownvalue.StringExact("circleci"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "circleci_organization.test_organization",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccOrganizationResourceConfig(name, vcs_type string) string {
	return fmt.Sprintf(`
resource "circleci_organization" "test_organization" {
  name 		= %[1]q
  vcs_type 	= %[2]q
}
`, name, vcs_type)
}
