// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"

	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	fakeContextToken          = "fake-context-token"
	contextDataSourceAddress  = "data.circleci_context.test"
	fakeContextRestrictionVal = "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
)

type fakeContextOrg struct {
	host       string
	orgID      string
	production fakecircle.Context
}

// newFakeContextOrg seeds an organization holding a "production" context with
// a restriction among more contexts than fit on one page of the list API (the
// fake sorts by name, so "production" is only on the third page), and a second
// organization with a context of the same name.
func newFakeContextOrg(t *testing.T) fakeContextOrg {
	t.Helper()

	fake := fakecircle.New(fakeContextToken)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	org, err := fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: "carta"})
	assert.NilError(t, err)
	for _, name := range []string{"development", "integration", "load-testing", "preview"} {
		_, err = fake.AddContext(fakecircle.NewContext{OrgID: org.ID, Name: name})
		assert.NilError(t, err)
	}
	production, err := fake.AddContext(fakecircle.NewContext{OrgID: org.ID, Name: "production"})
	assert.NilError(t, err)
	_, err = fake.AddContextRestriction(production.ID, fakecircle.NewContextRestriction{
		Name:  "All members",
		Type:  "group",
		Value: fakeContextRestrictionVal,
	})
	assert.NilError(t, err)

	other, err := fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: "other"})
	assert.NilError(t, err)
	_, err = fake.AddContext(fakecircle.NewContext{OrgID: other.ID, Name: "production"})
	assert.NilError(t, err)

	return fakeContextOrg{host: srv.URL + "/api/v2", orgID: org.ID.String(), production: production}
}

func testAccContextDataSourceConfig(host, body string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

data "circleci_context" "test" {
%[3]s
}
`, host, fakeContextToken, body)
}

func productionContextChecks(f fakeContextOrg) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(contextDataSourceAddress, tfjsonpath.New("id"),
			knownvalue.StringExact(f.production.ID.String())),
		statecheck.ExpectKnownValue(contextDataSourceAddress, tfjsonpath.New("name"),
			knownvalue.StringExact("production")),
		statecheck.ExpectKnownValue(contextDataSourceAddress, tfjsonpath.New("created_at"),
			knownvalue.StringFunc(func(v string) error {
				_, err := time.Parse(time.RFC3339, v)
				return err
			})),
		statecheck.ExpectKnownValue(contextDataSourceAddress, tfjsonpath.New("restrictions"),
			knownvalue.ListExact([]knownvalue.Check{
				knownvalue.ObjectPartial(map[string]knownvalue.Check{
					"name":  knownvalue.StringExact("All members"),
					"type":  knownvalue.StringExact("group"),
					"value": knownvalue.StringExact(fakeContextRestrictionVal),
				}),
			})),
	}
}

func TestAccContextDataSourceFake(t *testing.T) {
	f := newFakeContextOrg(t)

	t.Run("by name", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccContextDataSourceConfig(f.host, fmt.Sprintf(`
  organization_id = %q
  name            = "production"
`, f.orgID)),
					ConfigStateChecks: append(productionContextChecks(f),
						statecheck.ExpectKnownValue(contextDataSourceAddress, tfjsonpath.New("organization_id"),
							knownvalue.StringExact(f.orgID)),
					),
				},
			},
		})
	})

	t.Run("by id", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccContextDataSourceConfig(f.host, fmt.Sprintf(`
  id = %q
`, f.production.ID)),
					ConfigStateChecks: productionContextChecks(f),
				},
			},
		})
	})

	t.Run("name not found", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccContextDataSourceConfig(f.host, fmt.Sprintf(`
  organization_id = %q
  name            = "missing"
`, f.orgID)),
					ExpectError: regexp.MustCompile(
						`No context named "missing" in organization\s+` + regexp.QuoteMeta(f.orgID)),
				},
			},
		})
	})
}

func TestAccContextDataSourceFakeInvalidConfig(t *testing.T) {
	f := newFakeContextOrg(t)

	cases := []struct {
		name    string
		body    string
		wantErr *regexp.Regexp
	}{
		{
			name: "both id and name",
			body: fmt.Sprintf(`
  id              = %q
  organization_id = %q
  name            = "production"
`, f.production.ID, f.orgID),
			wantErr: regexp.MustCompile(`Invalid Attribute Combination`),
		},
		{
			name:    "neither id nor name",
			body:    "",
			wantErr: regexp.MustCompile(`Invalid Attribute Combination`),
		},
		{
			name: "name without organization_id",
			body: `
  name = "production"
`,
			wantErr: regexp.MustCompile(`Invalid Attribute Combination`),
		},
		{
			name: "organization_id with id",
			body: fmt.Sprintf(`
  id              = %q
  organization_id = %q
`, f.production.ID, f.orgID),
			wantErr: regexp.MustCompile(`Invalid Attribute Combination`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccContextDataSourceConfig(f.host, tc.body),
						ExpectError: tc.wantErr,
					},
				},
			})
		})
	}
}
