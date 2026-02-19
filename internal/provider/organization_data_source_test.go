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
						"data.circleci_organization.test_organization",
						tfjsonpath.New("id"),
						knownvalue.StringExact("3ddcf1d1-7f5f-4139-8cef-71ad0921a968"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_organization.test_organization",
						tfjsonpath.New("name"),
						knownvalue.StringExact("cci-terraform-test"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_organization.test_organization",
						tfjsonpath.New("slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_organization.test_organization",
						tfjsonpath.New("vcs_type"),
						knownvalue.StringExact("circleci"),
					),
				},
			},
		},
	})
}

const testOrganizationDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2"
}
  
data "circleci_organization" "test_organization" {
  id = "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
}
`
