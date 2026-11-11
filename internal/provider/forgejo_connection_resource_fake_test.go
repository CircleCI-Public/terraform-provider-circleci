// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/connection"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	fakeForgejoToken              = "fake-forgejo-token"
	fakeForgejoOrgID              = "7b1f8c2e-4d3a-4f6b-9e8d-1c2b3a4f5e6d"
	fakeForgejoConnectionAudience = "u:7:0f9e8d7c-6b5a-4c3d-2e1f-0a9b8c7d6e5f"
	fakeForgejoCheckoutAudience   = "u:7:1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
	fakeForgejoNewAudience        = "u:7:9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a"
	forgejoConnectionAddress      = "circleci_forgejo_connection.test"
)

type fakeForgejo struct {
	fake *fakecircle.Service
	host string
}

func newFakeForgejo(t *testing.T) fakeForgejo {
	t.Helper()
	fake := fakecircle.New(fakeForgejoToken)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return fakeForgejo{fake: fake, host: srv.URL + "/api/v2"}
}

func (f fakeForgejo) connections() *connection.Service {
	return connection.NewService(client.NewClient(f.host, fakeForgejoToken, "terraform-provider-circleci/test"))
}

func testAccForgejoConnectionConfig(host, checkoutAudience, headers, rotation string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_forgejo_connection" "test" {
  organization_id         = %[3]q
  url                     = "https://forgejo.example.com/"
  owner                   = "carta"
  connection_audience     = %[4]q
  checkout_audience       = %[5]q
  request_headers         = %[6]s
  webhook_secret_rotation = %[7]s
}
`, host, fakeForgejoToken, fakeForgejoOrgID, fakeForgejoConnectionAudience, checkoutAudience, headers, rotation)
}

func TestAccForgejoConnectionResource(t *testing.T) {
	f := newFakeForgejo(t)
	secretChanges := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccForgejoConnectionConfig(f.host, fakeForgejoCheckoutAudience,
					`{ "x-proxy-token" = "s3cret" }`, "null"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("owner_id"), knownvalue.StringExact("42")),
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("owner_type"), knownvalue.StringExact("organization")),
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("webhook_url"),
						knownvalue.StringRegexp(regexp.MustCompile(`^https://internal\.circleci\.com/private/soc/forgejo/webhook/`))),
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("webhook_events"),
						knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("push"), knownvalue.StringExact("pull_request")})),
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("request_headers"),
						knownvalue.MapExact(map[string]knownvalue.Check{"x-proxy-token": knownvalue.StringExact("s3cret")})),
					secretChanges.AddStateValue(forgejoConnectionAddress, tfjsonpath.New("webhook_secret")),
				},
			},
			{
				Config: testAccForgejoConnectionConfig(f.host, fakeForgejoNewAudience,
					`{ "x-proxy-token" = "s3cret", "x-other" = "v" }`, "null"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoConnectionAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: func(s *terraform.State) error {
					conn, err := f.connections().Get(context.TODO(), s.RootModule().Resources[forgejoConnectionAddress].Primary.ID)
					if err != nil {
						return err
					}
					if got := conn.Credentials.Access[connection.PurposeCheckout].Audience; got != fakeForgejoNewAudience {
						return fmt.Errorf("checkout audience is %q, want %q", got, fakeForgejoNewAudience)
					}
					if got := conn.Credentials.RequestHeaderNames; len(got) != 2 {
						return fmt.Errorf("request header names are %v, want two", got)
					}
					return nil
				},
			},
			{
				Config: testAccForgejoConnectionConfig(f.host, fakeForgejoNewAudience,
					`{ "x-proxy-token" = "s3cret", "x-other" = "v" }`, `"1"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoConnectionAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(forgejoConnectionAddress, tfjsonpath.New("webhook_secret")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					secretChanges.AddStateValue(forgejoConnectionAddress, tfjsonpath.New("webhook_secret")),
				},
			},
			{
				ResourceName:      forgejoConnectionAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fakeForgejoOrgID + "/" + s.RootModule().Resources[forgejoConnectionAddress].Primary.ID, nil
				},
				ImportStateVerifyIgnore: []string{
					"url",
					"owner",
					"request_headers",
					"webhook_url",
					"webhook_events",
					"webhook_secret",
					"webhook_secret_rotation",
				},
			},
		},
	})
}

func TestAccForgejoConnectionResourceHeaderDrift(t *testing.T) {
	f := newFakeForgejo(t)
	config := testAccForgejoConnectionConfig(f.host, fakeForgejoCheckoutAudience,
		`{ "x-proxy-token" = "s3cret" }`, "null")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					id := connectionIDFor(t, f)
					_, err := f.connections().SetRequestHeaders(context.TODO(), id, nil)
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoConnectionAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: func(s *terraform.State) error {
					conn, err := f.connections().Get(context.TODO(), s.RootModule().Resources[forgejoConnectionAddress].Primary.ID)
					if err != nil {
						return err
					}
					if got := conn.Credentials.RequestHeaderNames; len(got) != 1 || got[0] != "X-Proxy-Token" {
						return fmt.Errorf("request header names are %v, want [X-Proxy-Token]", got)
					}
					return nil
				},
			},
		},
	})
}

func TestAccForgejoConnectionResourceDeletedOutOfBand(t *testing.T) {
	f := newFakeForgejo(t)
	config := testAccForgejoConnectionConfig(f.host, fakeForgejoCheckoutAudience, "null", "null")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					err := f.connections().Delete(context.TODO(), connectionIDFor(t, f))
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoConnectionAddress, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccForgejoConnectionResourceRejected(t *testing.T) {
	f := newFakeForgejo(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccForgejoConnectionConfig(f.host, "not-an-audience", "null", "null"),
				ExpectError: regexp.MustCompile(`Attribute checkout_audience must be the audience`),
			},
			{
				Config: testAccForgejoConnectionConfig(f.host, fakeForgejoCheckoutAudience,
					`{ "X-Token" = "a", "x-token" = "b" }`, "null"),
				ExpectError: regexp.MustCompile(`Duplicate request header`),
			},
			{
				Config: testAccForgejoConnectionConfig(f.host,
					"u:8:1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d", "null", "null"),
				ExpectError: regexp.MustCompile(`same user`),
			},
		},
	})
}

// connectionIDFor finds the fake's only connection, for steps that change it
// behind Terraform's back.
func connectionIDFor(t *testing.T, f fakeForgejo) string {
	t.Helper()
	id := f.fake.ConnectionIDForOrg(fakeForgejoOrgID)
	assert.Assert(t, id != "", "no connection for org %s", fakeForgejoOrgID)
	return id
}

// Taking url and owner from the configuration after an import must update the
// connection in place rather than replace it.
func TestAccForgejoConnectionResourceImportUpdatesInPlace(t *testing.T) {
	f := newFakeForgejo(t)
	config := testAccForgejoConnectionConfig(f.host, fakeForgejoCheckoutAudience, "null", "null")

	var connID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					setup, err := f.connections().SetupForgejo(context.TODO(), fakeForgejoOrgID, connection.ForgejoSetup{
						URL:                "https://forgejo.example.com/",
						Owner:              "carta",
						ConnectionAudience: fakeForgejoConnectionAudience,
						CheckoutAudience:   fakeForgejoCheckoutAudience,
					})
					assert.NilError(t, err)
					connID = setup.ID
				},
				Config:             config,
				ResourceName:       forgejoConnectionAddress,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return fakeForgejoOrgID + "/" + connID, nil
				},
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(forgejoConnectionAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("id"), knownvalue.StringFunc(func(id string) error {
						if id != connID {
							return fmt.Errorf("connection was replaced: id %s, imported %s", id, connID)
						}
						return nil
					})),
					statecheck.ExpectKnownValue(forgejoConnectionAddress, tfjsonpath.New("owner"), knownvalue.StringExact("carta")),
				},
			},
		},
	})
}
