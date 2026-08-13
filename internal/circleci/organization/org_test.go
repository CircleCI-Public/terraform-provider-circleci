// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package organization_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/organization"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const testTok = "2d0a120d-0d44-40ae-906e-5856cc331f76"

func TestOrganizationService_Create(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	os := organization.NewOrganizationService(c)

	var org *organization.Organization
	t.Run("create", func(t *testing.T) {
		ctx := context.TODO()
		var err error
		org, err = os.Create(ctx, "test org name", "github")
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(org, &organization.Organization{
			Id:      "ignored",
			Name:    "test org name",
			VcsType: "github",
			Slug:    "github/test org name",
		}, cmpopts.IgnoreFields(organization.Organization{}, "Id")))
		assert.Check(t, org.Id != "")
	})

	t.Run("get", func(t *testing.T) {
		ctx := context.TODO()
		organization, err := os.Get(ctx, org.Id)
		assert.Assert(t, err)
		assert.Assert(t, organization.Id == org.Id)
	})

	t.Run("delete", func(t *testing.T) {
		ctx := context.TODO()
		err := os.Delete(ctx, org.Id)
		assert.Assert(t, err)
	})
}
