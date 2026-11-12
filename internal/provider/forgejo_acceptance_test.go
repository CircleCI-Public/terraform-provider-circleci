// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"terraform-provider-circleci/internal/circleci/connection"
)

// The live Forgejo tests need a Forgejo instance set up as the Forgejo guide
// describes, so they run only when CIRCLE_FORGEJO_URL is set. A CircleCI
// organization can have only one Forgejo connection, so these tests must never
// run concurrently against the same organization.
const (
	envForgejoURL                = "CIRCLE_FORGEJO_URL"
	envForgejoOwner              = "CIRCLE_FORGEJO_OWNER"
	envForgejoConnectionAudience = "CIRCLE_FORGEJO_CONNECTION_AUDIENCE"
	envForgejoCheckoutAudience   = "CIRCLE_FORGEJO_CHECKOUT_AUDIENCE"
	envForgejoRepo               = "CIRCLE_FORGEJO_REPO"
	envForgejoOtherRepo          = "CIRCLE_FORGEJO_OTHER_REPO"
	envForgejoOrgID              = "CIRCLE_FORGEJO_ORG_ID"
	envForgejoRequestHeaders     = "CIRCLE_FORGEJO_REQUEST_HEADERS"
)

type testAccForgejoEnv struct {
	url, owner, connectionAudience, checkoutAudience, repo, otherRepo, orgID string
	requestHeaders                                                           map[string]string
}

// testAccForgejoEnvFromOS reads the Forgejo test environment. Header values
// that are not valid JSON are left empty for testAccForgejoPreCheck to report.
func testAccForgejoEnvFromOS() testAccForgejoEnv {
	env := testAccForgejoEnv{
		url:                os.Getenv(envForgejoURL),
		owner:              os.Getenv(envForgejoOwner),
		connectionAudience: os.Getenv(envForgejoConnectionAudience),
		checkoutAudience:   os.Getenv(envForgejoCheckoutAudience),
		repo:               os.Getenv(envForgejoRepo),
		otherRepo:          os.Getenv(envForgejoOtherRepo),
		orgID:              os.Getenv(envForgejoOrgID),
		requestHeaders:     map[string]string{},
	}
	if env.orgID == "" {
		env.orgID = testAccRunnerOrgID
	}
	if raw := os.Getenv(envForgejoRequestHeaders); raw != "" {
		_ = json.Unmarshal([]byte(raw), &env.requestHeaders)
	}
	return env
}

// testAccForgejoPreCheck skips the test unless the Forgejo environment is
// configured, fails it when the configuration is incomplete, and removes any
// Forgejo connection a previous run left in the organization.
func testAccForgejoPreCheck(t *testing.T, env testAccForgejoEnv) {
	t.Helper()
	if env.url == "" {
		t.Skipf("Forgejo acceptance tests skipped unless %s is set", envForgejoURL)
	}
	testAccPreCheck(t)

	var missing []string
	for name, value := range map[string]string{
		envForgejoOwner:              env.owner,
		envForgejoConnectionAudience: env.connectionAudience,
		envForgejoCheckoutAudience:   env.checkoutAudience,
		envForgejoRepo:               env.repo,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%s is set, so these must be too: %s", envForgejoURL, strings.Join(missing, ", "))
	}
	if raw := os.Getenv(envForgejoRequestHeaders); raw != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(raw), &headers); err != nil {
			t.Fatalf("%s must be a JSON object of header names to values: %v", envForgejoRequestHeaders, err)
		}
	}

	testAccCleanupForgejoConnections(t, env.orgID)
}

// testAccCleanupForgejoConnections deletes the organization's Forgejo
// connection, which only a run that died before its destroy step leaves
// behind, since the tests never run concurrently. Keep no hand-made Forgejo
// connection in the test organization: this deletes it.
func testAccCleanupForgejoConnections(t *testing.T, orgID string) {
	t.Helper()

	svc := connection.NewService(testAccClient(t))
	ctx := context.Background()

	conns, err := svc.List(ctx, orgID)
	if err != nil {
		t.Fatalf("cleanup: listing the connections of organization %s: %v", orgID, err)
	}
	for _, c := range conns {
		if c.Provider != connection.ProviderForgejo {
			continue
		}
		if err := svc.Delete(ctx, c.ID); err != nil {
			t.Fatalf("cleanup: deleting leftover Forgejo connection %s: %v", c.ID, err)
		}
		t.Logf("cleanup: deleted leftover Forgejo connection %s to %s", c.ID, c.ProviderURL)
	}
}

// forgejoLiveOpts holds the values that differ between the steps of
// TestAccForgejoLive.
type forgejoLiveOpts struct {
	projectName string
	repo        string
	description string
	preset      string
	disabled    bool
	rotation    string
}

func testAccForgejoLiveConfig(env testAccForgejoEnv, o forgejoLiveOpts) string {
	return fmt.Sprintf(`
variable "request_headers" {
  type      = map(string)
  sensitive = true
}

resource "circleci_project" "forgejo" {
  name            = %[1]q
  organization_id = %[2]q
}

resource "circleci_forgejo_connection" "test" {
  organization_id         = %[2]q
  url                     = %[3]q
  owner                   = %[4]q
  connection_audience     = %[5]q
  checkout_audience       = %[6]q
  request_headers         = var.request_headers
  webhook_secret_rotation = %[11]q
}

resource "circleci_pipeline" "forgejo" {
  project_id                     = circleci_project.forgejo.id
  name                           = "build"
  description                    = %[8]q
  config_source_provider         = "forgejo"
  config_source_repo_full_name   = %[7]q
  config_source_file_path        = ".circleci/config.yml"
  checkout_source_provider       = "forgejo"
  checkout_source_repo_full_name = %[7]q

  depends_on = [circleci_forgejo_connection.test]
}

resource "circleci_trigger" "forgejo" {
  project_id                    = circleci_project.forgejo.id
  pipeline_id                   = circleci_pipeline.forgejo.id
  event_source_provider         = "forgejo"
  event_source_repo_external_id = circleci_pipeline.forgejo.checkout_source_repo_external_id
  event_preset                  = %[9]q
  disabled                      = %[10]t
}
`, o.projectName, env.orgID, env.url, env.owner, env.connectionAudience, env.checkoutAudience,
		o.repo, o.description, o.preset, o.disabled, o.rotation)
}

func TestAccForgejoLive(t *testing.T) {
	env := testAccForgejoEnvFromOS()
	projectName := testAccUniqueName("forgejo")
	const (
		conn    = "circleci_forgejo_connection.test"
		pipe    = "circleci_pipeline.forgejo"
		trig    = "circleci_trigger.forgejo"
		project = "circleci_project.forgejo"
	)
	secretChanges := statecheck.CompareValue(compare.ValuesDiffer())

	headerVars := make(map[string]config.Variable, len(env.requestHeaders))
	for name, value := range env.requestHeaders {
		headerVars[name] = config.StringVariable(value)
	}
	vars := config.Variables{"request_headers": config.MapVariable(headerVars)}

	createdOpts := forgejoLiveOpts{
		projectName: projectName,
		repo:        env.repo,
		description: "Created by the acceptance tests",
		preset:      "only-build-prs",
		rotation:    "0",
	}
	updatedOpts := forgejoLiveOpts{
		projectName: projectName,
		repo:        env.repo,
		description: "Updated by the acceptance tests",
		preset:      "all-pushes",
		disabled:    true,
		rotation:    "1",
	}
	movedOpts := updatedOpts
	movedOpts.repo = env.otherRepo

	created := testAccForgejoLiveConfig(env, createdOpts)
	updated := testAccForgejoLiveConfig(env, updatedOpts)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccForgejoPreCheck(t, env) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			conns, err := connection.NewService(testAccClient(t)).List(context.Background(), env.orgID)
			if err != nil {
				return err
			}
			for _, c := range conns {
				if c.Provider == connection.ProviderForgejo {
					return fmt.Errorf("forgejo connection %s still exists", c.ID)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				ConfigVariables: vars,
				Config:          created,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(conn, tfjsonpath.New("owner_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(conn, tfjsonpath.New("webhook_url"),
						knownvalue.StringRegexp(regexp.MustCompile(`/forgejo/webhook/`))),
					statecheck.ExpectKnownValue(pipe, tfjsonpath.New("checkout_source_repo_external_id"),
						knownvalue.NotNull()),
					statecheck.ExpectKnownValue(trig, tfjsonpath.New("event_preset"),
						knownvalue.StringExact("only-build-prs")),
					statecheck.CompareValuePairs(
						pipe, tfjsonpath.New("checkout_source_repo_external_id"),
						trig, tfjsonpath.New("event_source_repo_external_id"),
						compare.ValuesSame(),
					),
					secretChanges.AddStateValue(conn, tfjsonpath.New("webhook_secret")),
				},
			},
			{
				ConfigVariables: vars,
				Config:          updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(conn, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(pipe, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(trig, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(project, plancheck.ResourceActionNoop),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(pipe, tfjsonpath.New("description"),
						knownvalue.StringExact("Updated by the acceptance tests")),
					statecheck.ExpectKnownValue(trig, tfjsonpath.New("event_preset"), knownvalue.StringExact("all-pushes")),
					statecheck.ExpectKnownValue(trig, tfjsonpath.New("disabled"), knownvalue.Bool(true)),
					secretChanges.AddStateValue(conn, tfjsonpath.New("webhook_secret")),
				},
			},
			{
				ConfigVariables:   vars,
				Config:            updated,
				ResourceName:      conn,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(conn, "organization_id", "id"),
				ImportStateVerifyIgnore: []string{
					"url", "owner", "request_headers", "webhook_url", "webhook_events", "webhook_secret", "webhook_secret_rotation",
				},
			},
			{
				ConfigVariables:   vars,
				Config:            updated,
				ResourceName:      pipe,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(pipe, "project_id", "id"),
			},
			{
				ConfigVariables:   vars,
				Config:            updated,
				ResourceName:      trig,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(trig, "project_id", "id"),
			},
			{
				SkipFunc:        func() (bool, error) { return env.otherRepo == "", nil },
				ConfigVariables: vars,
				Config:          testAccForgejoLiveConfig(env, movedOpts),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(pipe, plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectResourceAction(trig, plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectResourceAction(conn, plancheck.ResourceActionNoop),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(pipe, tfjsonpath.New("checkout_source_repo_full_name"),
						knownvalue.StringFunc(func(name string) error {
							if !strings.EqualFold(name, env.otherRepo) {
								return fmt.Errorf("checkout repository is %s, want %s", name, env.otherRepo)
							}
							return nil
						})),
				},
			},
		},
	})
}
