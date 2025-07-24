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

func TestAccOrganizationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testOrganizationDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_organization.test_org",
						tfjsonpath.New("id"),
						knownvalue.StringExact("1302c657-19e5-4a82-93ed-366b37c9f403"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_organization.test_org",
						tfjsonpath.New("name"),
						knownvalue.StringExact("circleci-testing"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_organization.test_org",
						tfjsonpath.New("vcs_type"),
						knownvalue.StringExact("github"),
					),
				},
			},
		},
	})
}

const testOrganizationDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2/"
}
  
data "circleci_organization" "test_org" {
  id = "1302c657-19e5-4a82-93ed-366b37c9f403"
}
`
