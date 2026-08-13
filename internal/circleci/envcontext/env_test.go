// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package envcontext_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/envcontext"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const testTok = "0c3f30ae-66c3-40c9-9674-db6774f657fb"

func TestEnvService_List(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	envService := envcontext.NewEnvService(c)

	o, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "test org",
	})
	assert.Assert(t, err)
	orgCtx, err := fc.AddContext(fakecircle.NewContext{
		OrgID: o.ID,
		Name:  "test context",
	})
	assert.Assert(t, err)
	_, err = fc.AddContextEnv(orgCtx.ID, fakecircle.NewEnvVarContext{
		Variable: "FIREBASE_TOKEN",
	})
	assert.Assert(t, err)

	t.Run("list", func(t *testing.T) {
		ctx := context.TODO()
		envs, err := envService.List(ctx, orgCtx.ID.String())
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(envs, []envcontext.EnvVariable{
			{
				ContextId: orgCtx.ID.String(),
				Variable:  "FIREBASE_TOKEN",
				UpdatedAt: time.Now(),
				CreatedAt: time.Now(),
			},
		}, cmpopts.EquateApproxTime(time.Second)))
	})
}

func TestEnvService_Create(t *testing.T) {
	fc := fakecircle.New(testTok)
	srv := httptest.NewServer(fc)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test")
	envService := envcontext.NewEnvService(c)

	o, err := fc.AddOrg(fakecircle.NewOrg{
		Type: fakecircle.TypeCircleCI,
		Name: "test org",
	})
	assert.Assert(t, err)
	orgCtx, err := fc.AddContext(fakecircle.NewContext{
		OrgID: o.ID,
		Name:  "test context",
	})
	assert.Assert(t, err)

	t.Run("empty", func(t *testing.T) {
		ctx := context.TODO()
		envs, err := envService.List(ctx, orgCtx.ID.String())
		assert.Assert(t, err)
		assert.Check(t, cmp.Len(envs, 0))
	})

	t.Run("create", func(t *testing.T) {
		ctx := context.TODO()
		envCreated, err := envService.Create(ctx, orgCtx.ID.String(), "VALUE", "test_sdk")
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(envCreated, &envcontext.EnvVariable{
			ContextId: orgCtx.ID.String(),
			Variable:  "test_sdk",
			UpdatedAt: time.Now(),
			CreatedAt: time.Now(),
		}, cmpopts.EquateApproxTime(time.Second)))
	})

	t.Run("list", func(t *testing.T) {
		ctx := context.TODO()
		envs, err := envService.List(ctx, orgCtx.ID.String())
		assert.Assert(t, err)
		assert.Check(t, cmp.DeepEqual(envs, []envcontext.EnvVariable{
			{
				ContextId: orgCtx.ID.String(),
				Variable:  "test_sdk",
				UpdatedAt: time.Now(),
				CreatedAt: time.Now(),
			},
		}, cmpopts.EquateApproxTime(time.Second)))
	})

	t.Run("delete", func(t *testing.T) {
		ctx := context.TODO()
		err := envService.Delete(ctx, orgCtx.ID.String(), "test_sdk")
		assert.Assert(t, err)
	})

	t.Run("empty again", func(t *testing.T) {
		ctx := context.TODO()
		envs, err := envService.List(ctx, orgCtx.ID.String())
		assert.Assert(t, err)
		assert.Check(t, cmp.Len(envs, 0))
	})
}
