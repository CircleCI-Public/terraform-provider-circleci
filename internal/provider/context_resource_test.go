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

func TestAccContextResource(t *testing.T) {
	dateRegex, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$`)
	if err != nil {
		t.Fatal("Could not create Date Regex for testing.")
	}
	randName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccContextResourceConfig(randName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_context.test_context",
						tfjsonpath.New("name"),
						knownvalue.StringExact(randName),
					),
					statecheck.ExpectKnownValue(
						"circleci_context.test_context",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact("3ddcf1d1-7f5f-4139-8cef-71ad0921a968"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context.test_context",
						tfjsonpath.New("created_at"),
						knownvalue.StringRegexp(dateRegex),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccContextResourceConfig(name string) string {
	return fmt.Sprintf(`
  resource "circleci_context" "test_context" {
  name            = %[1]q
  organization_id = "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
}
`, name)
}
