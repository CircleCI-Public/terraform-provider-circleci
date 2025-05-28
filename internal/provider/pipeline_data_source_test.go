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
						knownvalue.StringExact("fefb451c-9966-4b75-b555-d4d94d7116ef"),
					),
					statecheck.ExpectKnownValue(
						"data.circleci_pipeline.test_pipeline",
						tfjsonpath.New("project_id"),
						knownvalue.StringExact("61169e84-93ee-415d-8d65-ddf6dc0d2939"),
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
  id         = "fefb451c-9966-4b75-b555-d4d94d7116ef"
  project_id = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
}
`
