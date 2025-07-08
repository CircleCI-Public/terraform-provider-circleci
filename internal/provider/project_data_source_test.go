// Copyright (c) HashiCorp, Inc.
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

func TestAccProjectDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testProjectDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("id"),
						knownvalue.StringExact("e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-project"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("auto_cancel_builds"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("disable_ssh"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("forks_receive_secret_env_vars"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("oss"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("set_github_status"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("setup_workflows"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("write_settings_requires_admin"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("pr_only_branch_overrides"),
						knownvalue.ListExact(
							[]knownvalue.Check{
								0: knownvalue.StringExact("main"),
							},
						),
					),
				},
			},
		},
	})
}

const testProjectDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2/"
}

data "circleci_project" "test_project" {
  slug = "circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"
}
`
