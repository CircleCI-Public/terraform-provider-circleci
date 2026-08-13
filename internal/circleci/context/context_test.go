// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package context_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	sdkcontext "terraform-provider-circleci/internal/circleci/context"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const testTok = "9708df71-aced-497e-b9d0-f12837c72492"

func TestContextService_List(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	contextService := sdkcontext.NewContextService(c)

	var o fakecircle.Org
	var orgCtx fakecircle.Context
	t.Run("add_org", func(t *testing.T) {
		var err error
		o, err = fc.AddOrg(fakecircle.NewOrg{
			Type: fakecircle.TypeCircleCI,
			Name: "test org",
		})
		assert.Assert(t, err)
		orgCtx, err = fc.AddContext(fakecircle.NewContext{
			OrgID: o.ID,
			Name:  "test context",
		})
		assert.Assert(t, err)
	})

	t.Run("add_other_org", func(t *testing.T) {
		o2, err := fc.AddOrg(fakecircle.NewOrg{
			Type: fakecircle.TypeCircleCI,
			Name: "other",
		})
		assert.Assert(t, err)
		_, err = fc.AddContext(fakecircle.NewContext{
			OrgID: o2.ID,
			Name:  "other test context",
		})
		assert.Assert(t, err)
	})

	t.Run("list", func(t *testing.T) {
		ctx := context.TODO()
		ctxs, err := contextService.List(ctx, o.Slug)
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(ctxs, []sdkcontext.Context{
			{
				ID:        orgCtx.ID.String(),
				Name:      "test context",
				CreatedAt: "ignored",
			},
		}, cmpopts.IgnoreFields(sdkcontext.Context{}, "CreatedAt")))
	})
}

func TestContextService_Get(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	contextService := sdkcontext.NewContextService(c)

	var o fakecircle.Org
	var orgCtx fakecircle.Context
	t.Run("add_org", func(t *testing.T) {
		var err error
		o, err = fc.AddOrg(fakecircle.NewOrg{
			Type: fakecircle.TypeCircleCI,
			Name: "8e4z1Akd74woxagxnvLT5q",
		})
		assert.Assert(t, err)
		orgCtx, err = fc.AddContext(fakecircle.NewContext{
			OrgID: o.ID,
			Name:  "test context",
		})
		assert.Assert(t, err)
	})

	t.Run("get", func(t *testing.T) {
		ctx := context.TODO()
		r, err := contextService.Get(ctx, orgCtx.ID.String())
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(r, &sdkcontext.Context{
			ID:        orgCtx.ID.String(),
			Name:      "test context",
			CreatedAt: "ignored",
		}, cmpopts.IgnoreFields(sdkcontext.Context{}, "CreatedAt")))
	})
}

func TestContextService_Full(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	contextService := sdkcontext.NewContextService(c)

	var o fakecircle.Org
	t.Run("add_org", func(t *testing.T) {
		var err error
		o, err = fc.AddOrg(fakecircle.NewOrg{
			Type: fakecircle.TypeCircleCI,
			Name: "8e4z1Akd74woxagxnvLT5q",
		})
		assert.Assert(t, err)
	})

	var ctxCreated *sdkcontext.Context
	assert.Assert(t, t.Run("create", func(t *testing.T) {
		ctx := context.TODO()
		var err error
		ctxCreated, err = contextService.Create(ctx, o.ID.String(), "Test ctx")
		assert.Assert(t, err)
	}))

	t.Run("get", func(t *testing.T) {
		ctx := context.TODO()
		orgCtx, err := contextService.Get(ctx, ctxCreated.ID)
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(orgCtx, &sdkcontext.Context{
			ID:        "ignored",
			Name:      "Test ctx",
			CreatedAt: "ignored",
		}, cmpopts.IgnoreFields(sdkcontext.Context{}, "ID", "CreatedAt")))
		assert.Check(t, orgCtx.ID != "")
		assert.Check(t, orgCtx.CreatedAt != "")
	})

	t.Run("delete", func(t *testing.T) {
		ctx := context.TODO()
		err := contextService.Delete(ctx, ctxCreated.ID)
		assert.Assert(t, err)
	})

	t.Run("get", func(t *testing.T) {
		ctx := context.TODO()
		ctxFetched, err := contextService.Get(ctx, ctxCreated.ID)
		assert.Assert(t, cmp.ErrorContains(err, "context not found"))
		assert.Check(t, cmp.Nil(ctxFetched))
	})
}
