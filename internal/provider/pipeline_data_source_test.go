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

func TestAccPipelineDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testPipelineDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.circleci_pipeline.test_pipeline",
						tfjsonpath.New("id"),
						knownvalue.StringExact("03d415e5-0ff5-5a09-afb2-b635a6f53bed"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_pipeline.test_pipeline",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("4339ead9-cbd3-4006-a3cf-8253db6c9a77"),
					),
				},
			},
		},
	})
}

const testPipelineDataSourceConfig = `
provider "circleci" {
  host = "https://circleci.com/api/v2/"
}

data "circleci_pipeline" "test_pipeline" {
  id         = "03d415e5-0ff5-5a09-afb2-b635a6f53bed"
  project_id = "4339ead9-cbd3-4006-a3cf-8253db6c9a77"
}

output "circleci_pipeline_id" {
  value = data.circleci_pipeline.test_pipeline.id
}

output "circleci_pipeline_project_id" {
  value = data.circleci_pipeline.test_pipeline.project_id
}

output "circleci_pipeline_name" {
  value = data.circleci_pipeline.test_pipeline.name
}

output "circleci_pipeline_description" {
  value = data.circleci_pipeline.test_pipeline.description
}

output "circleci_pipeline_created_at" {
  value = data.circleci_pipeline.test_pipeline.created_at
}

output "circleci_pipeline_config_source_provider" {
  value = data.circleci_pipeline.test_pipeline.config_source_provider
}

output "circleci_pipeline_config_source_file_path" {
  value = data.circleci_pipeline.test_pipeline.config_source_file_path
}

output "circleci_pipeline_config_source_repo_full_name" {
  value = data.circleci_pipeline.test_pipeline.config_source_repo_full_name
}

output "circleci_pipeline_config_source_repo_external_id" {
  value = data.circleci_pipeline.test_pipeline.config_source_repo_external_id
}

output "circleci_pipeline_checkout_source_provider" {
  value = data.circleci_pipeline.test_pipeline.checkout_source_provider
}

output "circleci_pipeline_checkout_source_repo_full_name" {
  value = data.circleci_pipeline.test_pipeline.checkout_source_repo_full_name
}

output "circleci_pipeline_checkout_source_repo_external_id" {
  value = data.circleci_pipeline.test_pipeline.checkout_source_repo_external_id
}
`
