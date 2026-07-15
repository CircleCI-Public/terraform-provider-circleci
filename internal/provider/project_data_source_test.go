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
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact("3ddcf1d1-7f5f-4139-8cef-71ad0921a968"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("organization_name"),
						knownvalue.StringExact("cci-terraform-test"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("organization_slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("slug"),
						knownvalue.StringExact("circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("vcs_info").AtMapKey("default_branch"),
						knownvalue.StringExact("main"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("vcs_info").AtMapKey("provider"),
						knownvalue.StringExact("CircleCI"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_project.test_project",
						tfjsonpath.New("vcs_info").AtMapKey("vcs_url"),
						knownvalue.StringExact("//circleci.com/3ddcf1d1-7f5f-4139-8cef-71ad0921a968/e2e8ae23-57dc-4e95-bc67-633fdeb4ac33"),
					),
				},
			},
		},
	})
}

const testProjectDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2"
}

data "circleci_project" "test_project" {
  slug = "circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"
}
`

func TestProjectDataSourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := datasource.SchemaRequest{}
	schemaResponse := &datasource.SchemaResponse{}

	NewProjectDataSource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}
