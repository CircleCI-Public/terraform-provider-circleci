// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// newFakeTriggerServer serves a single in-memory trigger, mimicking the two
// behaviours of the real API that the update path depends on: PATCH rejects the
// create-only event_source.repo and event_source.provider fields, and refs that
// were never set are reported as "" rather than omitted. Serving these locally
// keeps the trigger update paths covered without an API token.
func newFakeTriggerServer(t *testing.T) string {
	t.Helper()

	var (
		mu      sync.Mutex
		trigger = map[string]any{}
	)

	// write always includes the ref fields, as the real API does.
	write := func(w http.ResponseWriter) {
		out := map[string]any{"checkout_ref": "", "config_ref": "", "name": "", "event_name": ""}
		for k, v := range trigger {
			out[k] = v
		}
		_ = json.NewEncoder(w).Encode(out)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)

		switch r.Method {
		case http.MethodPost:
			_ = json.Unmarshal(body, &trigger)
			trigger["id"] = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			trigger["created_at"] = "2026-01-01T00:00:00Z"
			write(w)

		case http.MethodPatch:
			var updates map[string]any
			_ = json.Unmarshal(body, &updates)

			if eventSource, ok := updates["event_source"].(map[string]any); ok {
				for _, createOnly := range []string{"repo", "provider"} {
					if _, present := eventSource[createOnly]; present {
						w.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(w).Encode(map[string]string{
							"message": fmt.Sprintf("Unexpected field 'event_source.%s'", createOnly),
						})
						return
					}
				}
			}
			for field, value := range updates {
				if field == "event_source" {
					continue
				}
				trigger[field] = value
			}
			write(w)

		case http.MethodGet:
			write(w)

		case http.MethodDelete:
			trigger = map[string]any{}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Trigger deleted."})

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

func testAccTriggerResourceGithubAppConfigNoRefs(host string, disabled bool) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = "fake"
}

resource "circleci_trigger" "test_trigger_no_refs" {
  project_id                    = "00000000-0000-0000-0000-000000000001"
  pipeline_id                   = "00000000-0000-0000-0000-000000000002"
  event_source_provider         = "github_app"
  event_source_repo_external_id = "1028689591"
  event_preset                  = "all-pushes"
  disabled                      = %[2]t
}
`, host, disabled)
}

func testAccTriggerResourceGithubAppConfigNoRefsNoRepoExternalId(host string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = "fake"
}

resource "circleci_trigger" "test_trigger_no_refs" {
  project_id            = "00000000-0000-0000-0000-000000000001"
  pipeline_id           = "00000000-0000-0000-0000-000000000002"
  event_source_provider = "github_app"
  event_preset          = "all-pushes"
  disabled              = false
}
`, host)
}

// A github_app trigger that omits checkout_ref and config_ref, which the docs
// require when the event source repo matches the pipeline's, must support
// in-place updates. Update previously copied the API's "" over the null plan
// values, which fails Terraform's provider consistency check.
func TestAccTriggerResourceGithubAppNoRefsUpdate(t *testing.T) {
	host := newFakeTriggerServer(t)

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
			knownvalue.StringExact("1028689591"),
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
	host := newFakeTriggerServer(t)

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
