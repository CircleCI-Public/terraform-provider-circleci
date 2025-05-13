// Copyright (c) HashiCorp, Inc.
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

func TestAccPipelineResource(t *testing.T) {
	t.Skip()
	uuidRegex, err := regexp.Compile(`[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}`)
	if err != nil {
		t.Fatalf("Regex to check UUID could not be created")
	}
	dateRegex, err := regexp.Compile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)
	if err != nil {
		t.Fatal("Could not create Date Regex for testing.")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPipelineResourceConfig("organization_id", "pipeline_name"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact("organization_id"),
					),
					statecheck.ExpectKnownValue(
						"circleci_pipeline.test_pipeline",
						tfjsonpath.New("name"),
						knownvalue.StringExact("pipeline_name"),
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
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPipelineResourceConfig(organization_id, name string) string {
	return fmt.Sprintf(`
resource "circleci_pipeline" "test_pipeline" {
	organization_id = %[1]q
	name = %[2]q
}
`, organization_id, name)
}
