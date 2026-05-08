// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProvider_RunnerHostHCL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `{"items":[{"id":"11111111-2222-3333-4444-555555555555","resource_class":"ns/rc","description":""}]}`)
	}))
	t.Cleanup(srv.Close)

	cfg := fmt.Sprintf(`
provider "circleci" {
  host        = "http://127.0.0.1:1"
  runner_host = %q
  key         = "fake"
}
data "circleci_runner_resource_class" "t" {
  organization_id = "00000000-1111-2222-3333-444444444444"
  resource_class  = "ns/rc"
}
`, srv.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: cfg}},
	})
}
