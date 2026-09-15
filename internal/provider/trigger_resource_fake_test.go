// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	fakeTriggerToken          = "fake-token"
	fakeTriggerProjectID      = "00000000-0000-0000-0000-000000000001"
	fakeTriggerPipelineDefID  = "00000000-0000-0000-0000-000000000002"
	fakeTriggerRepoExternalID = "1028689591"
)

// newFakeTriggerHost serves the in-memory CircleCI fake and returns a host the
// provider can be pointed at, so the trigger paths can be exercised without an
// API token.
func newFakeTriggerHost(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(fakecircle.New(fakeTriggerToken))
	t.Cleanup(srv.Close)

	return srv.URL + "/api/v2"
}

func testAccTriggerResourceGithubAppConfigNoRefs(host string, disabled bool) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_trigger" "test_trigger_no_refs" {
  project_id                    = %[3]q
  pipeline_id                   = %[4]q
  event_source_provider         = "github_app"
  event_source_repo_external_id = %[5]q
  event_preset                  = "all-pushes"
  disabled                      = %[6]t
}
`, host, fakeTriggerToken, fakeTriggerProjectID, fakeTriggerPipelineDefID, fakeTriggerRepoExternalID, disabled)
}

func testAccTriggerResourceGithubAppConfigNoRefsNoRepoExternalId(host string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_trigger" "test_trigger_no_refs" {
  project_id            = %[3]q
  pipeline_id           = %[4]q
  event_source_provider = "github_app"
  event_preset          = "all-pushes"
  disabled              = false
}
`, host, fakeTriggerToken, fakeTriggerProjectID, fakeTriggerPipelineDefID)
}

// A github_app trigger that omits checkout_ref and config_ref, which the docs
// require when the event source repo matches the pipeline definition's, must
// support in-place updates. Update previously copied the API's "" over the null
// plan values, which fails Terraform's provider consistency check.
func TestAccTriggerResourceGithubAppNoRefsUpdate(t *testing.T) {
	host := newFakeTriggerHost(t)

	refsStayNull := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(
			"circleci_trigger.test_trigger_no_refs",
			tfjsonpath.New("checkout_ref"),
			knownvalue.Null(),
		),
		statecheck.ExpectKnownValue(
			"circleci_trigger.test_trigger_no_refs",
			tfjsonpath.New("config_ref"),
			knownvalue.Null(),
		),
		statecheck.ExpectKnownValue(
			"circleci_trigger.test_trigger_no_refs",
			tfjsonpath.New("event_source_repo_external_id"),
			knownvalue.StringExact(fakeTriggerRepoExternalID),
		),
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config:            testAccTriggerResourceGithubAppConfigNoRefs(host, false),
				ConfigStateChecks: refsStayNull,
			},
			// Update and Read testing
			{
				Config: testAccTriggerResourceGithubAppConfigNoRefs(host, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"circleci_trigger.test_trigger_no_refs",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				ConfigStateChecks: append(refsStayNull,
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_no_refs",
						tfjsonpath.New("disabled"),
						knownvalue.Bool(true),
					),
				),
			},
		},
	})
}

// Removing event_source_repo_external_id must report the provider's validation
// error as an in-place update. Planning a replacement destroys the trigger
// before Create rejects the configuration, leaving nothing behind.
func TestAccTriggerResourceGithubAppRemoveRepoExternalIdUpdatesInPlace(t *testing.T) {
	host := newFakeTriggerHost(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccTriggerResourceGithubAppConfigNoRefs(host, false)},
			{
				Config: testAccTriggerResourceGithubAppConfigNoRefsNoRepoExternalId(host),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"circleci_trigger.test_trigger_no_refs",
							plancheck.ResourceActionUpdate,
						),
					},
				},
				ExpectError: regexp.MustCompile(`requires[\s]+event_source_repo_external_id`),
			},
		},
	})
}
