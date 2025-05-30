// Copyright (c) Copyright (c) CircleCI and HashiCorp, Inc.
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

func TestAccContextRestrictionResource(t *testing.T) {
	uuidRegex, err := regexp.Compile(`[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`)
	if err != nil {
		t.Fatalf("Regex to check UUID could not be created")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccContextRestrictionResourceConfig("project", "7d4d46da-49d1-4b3a-9a1b-3356ddfa67d6"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("context_id"),
						knownvalue.StringExact("e51158a2-f59c-4740-9eb4-d20609baa07e"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("type"),
						knownvalue.StringExact("project"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("value"),
						knownvalue.StringExact("7d4d46da-49d1-4b3a-9a1b-3356ddfa67d6"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("7d4d46da-49d1-4b3a-9a1b-3356ddfa67d6"),
					),
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("name"),
						knownvalue.StringExact("david"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccContextRestrictionResourceConfig(sometype, value string) string {
	return fmt.Sprintf(`
data "circleci_context" "test_context" {
  id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
}

resource "circleci_context_restriction" "test_context_restriction" {
	context_id = data.circleci_context.test_context.id
	type = %[1]q
	value = %[2]q
}
`, sometype, value)
}
