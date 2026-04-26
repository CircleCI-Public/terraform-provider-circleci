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

func TestAccPipelineResource(t *testing.T) {
	uuidRegex, err := regexp.Compile(`[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`)
	if err != nil {
		t.Fatalf("Regex to check UUID could not be created")
	}
	dateRegex, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$`)
	if err != nil {
		t.Fatal("Could not create Date Regex for testing.")
	}
	pipelineName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPipelineResourceConfig("61169e84-93ee-415d-8d65-ddf6dc0d2939", pipelineName, "original description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("name"),
						knownvalue.StringExact(pipelineName),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("created_at"),
						knownvalue.StringRegexp(dateRegex),
					),
				},
			},
			{
				Config: testAccPipelineResourceConfig("61169e84-93ee-415d-8d65-ddf6dc0d2939", pipelineName, "updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("name"),
						knownvalue.StringExact(pipelineName),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("description"),
						knownvalue.StringExact("updated description"),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("created_at"),
						knownvalue.StringRegexp(dateRegex),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "circleci_pipeline.test_pipeline",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					pipelineId, found := s.RootModule().Resources["circleci_pipeline.test_pipeline"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_pipeline.test_pipeline.id not found")
					}
					projectId, found := s.RootModule().Resources["circleci_pipeline.test_pipeline"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute circleci_pipeline.test_pipeline.project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectId, pipelineId), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPipelineResourceGithubServer(t *testing.T) {
	uuidRegex, err := regexp.Compile(`[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`)
	if err != nil {
		t.Fatalf("Regex to check UUID could not be created")
	}
	dateRegex, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$`)
	if err != nil {
		t.Fatal("Could not create Date Regex for testing.")
	}
	pipelineName := rand.Text()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPipelineResourceGithubServerConfig("20209578-aa1c-4b4c-9ca5-f6e38a47cf73", pipelineName, "original description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline_github_server",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("20209578-aa1c-4b4c-9ca5-f6e38a47cf73"),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline_github_server",
						tfjsonpath.New("name"),
						knownvalue.StringExact(pipelineName),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline_github_server",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(uuidRegex),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline_github_server",
						tfjsonpath.New("created_at"),
						knownvalue.StringRegexp(dateRegex),
					),
				},
			},
			{
				Config: testAccPipelineResourceGithubServerConfig("20209578-aa1c-4b4c-9ca5-f6e38a47cf73", pipelineName, "updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline_github_server",
						tfjsonpath.New("description"),
						knownvalue.StringExact("updated description"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "circleci_pipeline.test_pipeline_github_server",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					pipelineId, found := s.RootModule().Resources["circleci_pipeline.test_pipeline_github_server"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute circleci_pipeline.test_pipeline_github_server.id not found")
					}
					projectId, found := s.RootModule().Resources["circleci_pipeline.test_pipeline_github_server"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute circleci_pipeline.test_pipeline_github_server.project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectId, pipelineId), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPipelineResourceGithubServerConfig(project_id, name, description string) string {
	return fmt.Sprintf(`
resource "circleci_pipeline" "test_pipeline_github_server" {
	project_id = %[1]q
	name = %[2]q
	description = %[3]q
	config_source_provider = "github_server"
	config_source_file_path = "config_source_file_path"
	config_source_repo_external_id = "2259"
	checkout_source_provider = "github_server"
	checkout_source_repo_external_id = "2259"
}
`, project_id, name, description)
}

func testAccPipelineResourceConfig(project_id, name, description string) string {
	return fmt.Sprintf(`
resource "circleci_pipeline" "test_pipeline" {
	project_id = %[1]q
	name = %[2]q
	description = %[3]q
	config_source_provider = "github_app"
	config_source_file_path = "config_source_file_path"
	//config_source_repo_full_name = "cci-terraform-test/test-repo"
	config_source_repo_external_id = "952038793"
	checkout_source_provider = "github_app"
	//checkout_source_repo_full_name = "cci-terraform-test/test-repo"
	checkout_source_repo_external_id = "952038793"
}
`, project_id, name, description)
}
