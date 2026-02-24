// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
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

func TestAccRunnerTokenResource(t *testing.T) {
	resourceClass := "cci-terraform-test/acc-test-runner"
	nickname := "acc-test-token"
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRunnerTokenConfig(resourceClass, nickname),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_runner_token.test",
						tfjsonpath.New("resource_class"),
						knownvalue.StringExact(resourceClass),
					),
					statecheck.ExpectKnownValue(
						"circleci_runner_token.test",
						tfjsonpath.New("nickname"),
						knownvalue.StringExact(nickname),
					),
					statecheck.ExpectKnownValue(
						"circleci_runner_token.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					// token is sensitive but should be non-empty after create
					statecheck.ExpectKnownValue(
						"circleci_runner_token.test",
						tfjsonpath.New("token"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing — token value will be empty after import since it's write-once
			{
				ResourceName:            "circleci_runner_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rc, found := s.RootModule().Resources["circleci_runner_token.test"].Primary.Attributes["resource_class"]
					if !found {
						return "", errors.New("attribute resource_class not found")
					}
					id, found := s.RootModule().Resources["circleci_runner_token.test"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute id not found")
					}
					return fmt.Sprintf("%s/%s", rc, id), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRunnerTokenConfig(resourceClass, nickname string) string {
	return fmt.Sprintf(`
resource "circleci_runner_token" "test" {
  resource_class = %[1]q
  nickname       = %[2]q
}
`, resourceClass, nickname)
}
