// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccContextRestrictionResource(t *testing.T) {
	uuidRegex, err := regexp.Compile(`[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`)
	if err != nil {
		t.Fatalf("Regex to check UUID could not be created")
	}
	// The restriction's identity is (context, project), so this test creates its
	// own context rather than restricting the shared static one — otherwise two
	// concurrent terraform-version jobs create the same restriction.
	contextName := rand.Text()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccContextRestrictionResourceConfig(contextName, "project", "7d4d46da-49d1-4b3a-9a1b-3356ddfa67d6"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_context_restriction.test_context_restriction",
						tfjsonpath.New("context_id"),
						knownvalue.StringRegexp(uuidRegex),
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
					/*
						statecheck.ExpectKnownValue(
							"circleci_context_restriction.test_context_restriction",
							tfjsonpath.New("name"),
							knownvalue.StringExact("All members"),
						),
					*/
				},
			},
			// ImportState testing
			{
				ResourceName:            "circleci_context_restriction.test_context_restriction",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					// 1. Get the computed 'id' (context ID)
					contextID, found := s.RootModule().Resources["circleci_context_restriction.test_context_restriction"].Primary.Attributes["context_id"]
					if !found {
						return "", errors.New("attribute circleci_contexcircleci_context_restrictiont.test_context_restriction.context_id not found")
					}

					// 2. Get the known 'organization_id'
					restrictionID, found := s.RootModule().Resources["circleci_context_restriction.test_context_restriction"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_context_restriction.test_context_restriction.id not found")
					}

					// 3. Return the composite ID string: "RESTRICTION_ID/CONTEXT_ID"
					return fmt.Sprintf("%s/%s", contextID, restrictionID), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccContextRestrictionResourceConfig(contextName, sometype, value string) string {
	return fmt.Sprintf(`
resource "circleci_context" "test_context" {
  name            = %[1]q
  organization_id = "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
}

resource "circleci_context_restriction" "test_context_restriction" {
	context_id = circleci_context.test_context.id
	type = %[2]q
	value = %[3]q
}
`, contextName, sometype, value)
}
