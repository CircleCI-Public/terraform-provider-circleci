// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectSettingsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testProjectSettingsDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("auto_cancel_builds"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("build_fork_prs"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("disable_ssh"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("forks_receive_secret_env_vars"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("oss"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("set_github_status"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("setup_workflows"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("write_settings_requires_admin"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project_settings.test_project",
						tfjsonpath.New("pr_only_branch_overrides"),
						knownvalue.SetExact(
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

const testProjectSettingsDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2"
}

data "circleci_project_settings" "test_project" {
  slug = "circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"
}
`

func TestProjectSettingsDataSourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := datasource.SchemaRequest{}
	schemaResponse := &datasource.SchemaResponse{}

	NewProjectSettingsDataSource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}
