// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCircleCiProjectResource(t *testing.T) {
	projectName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig(
					projectName,
					"3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
					true,
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
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
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("auto_cancel_builds"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("pr_only_branch_overrides"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "circleci_project.test_project",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					slug, found := s.RootModule().Resources["circleci_project.test_project"].Primary.Attributes["slug"]
					if !found {
						return "", errors.New("attribute circleci_project.test_project.slug not found")
					}
					return slug, nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccGithubProjectResource(t *testing.T) {
	t.Skip()
	//projectName := "terraform-provider-test"
	//orgId := "ec6887ec-7d44-4b31-b468-7e552408ee32"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			/*{
				Config: testAccProjectResourceConfig(
					projectName,
					orgId,
					false,
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_slug"),
						knownvalue.StringExact("gh/david-montano-circleci"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgId),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(true),
					),
				},
			},*/
			// ImportState testing
			{
				ResourceName:      "circleci_project.test_project",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					slug, found := s.RootModule().Resources["circleci_project.test_project"].Primary.Attributes["slug"]
					if !found {
						return "", errors.New("attribute circleci_project.test_project.slug not found")
					}
					return slug, nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccCircleCiProjectOrgUpdateResource(t *testing.T) {
	t.Skip()
	projectName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig(
					projectName,
					"3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
					true,
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
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
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("auto_cancel_builds"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("pr_only_branch_overrides"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
			{
				Config: testAccProjectResourceConfig(
					projectName,
					"ec6887ec-7d44-4b31-b468-7e552408ee32",
					true,
					false,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_slug"),
						knownvalue.StringExact("circleci/WkZnacg2YgztDh3uKUsRPD"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact("ec6887ec-7d44-4b31-b468-7e552408ee32"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("auto_cancel_builds"),
						knownvalue.Bool(true),
					),
					/*
						statecheck.ExpectKnownValue(
							"circleci_project.build_fork_prs",
							tfjsonpath.New("build_fork_prs"),
							knownvalue.Bool(false),
						),
					*/
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("pr_only_branch_overrides"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "circleci_project.test_project",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					slug, found := s.RootModule().Resources["circleci_project.test_project"].Primary.Attributes["slug"]
					if !found {
						return "", errors.New("attribute circleci_project.test_project.slug not found")
					}
					return slug, nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccGithubProjectOrgUpdateResource(t *testing.T) {
	t.Skip()
	projectName := rand.Text()
	orgId := "ec6887ec-7d44-4b31-b468-7e552408ee32"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig(
					projectName,
					orgId,
					false,
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_slug"),
						knownvalue.StringExact("gh/david-montano-circleci"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgId),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				Config: testAccProjectResourceConfig(
					projectName,
					orgId,
					true,
					false,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_slug"),
						knownvalue.StringExact("gh/david-montano-circleci"),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgId),
					),
					statecheck.ExpectKnownValue(
						"circleci_project.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(false),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "circleci_project.test_project",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					slug, found := s.RootModule().Resources["circleci_project.test_project"].Primary.Attributes["slug"]
					if !found {
						return "", errors.New("attribute circleci_project.test_project.slug not found")
					}
					return slug, nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfig(name, organization_id string, auto_cancel_builds bool, build_forked_prs bool) string {
	return fmt.Sprintf(`
resource "circleci_project" "test_project" {
  name 				 = %[1]q
  organization_id 	 = %[2]q
  auto_cancel_builds = %[3]t 
  build_fork_prs     = %[4]t
}
`, name, organization_id, auto_cancel_builds, build_forked_prs)
}
