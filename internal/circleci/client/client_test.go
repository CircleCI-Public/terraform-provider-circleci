// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

func TestClient_RequestHelper(t *testing.T) {
	const testTok = "CCIPAT_865d543e-9d33-4157-a6cc-8f4416a02df0"

	fs := fakecircle.New(testTok)
	srv := httptest.NewServer(fs)
	t.Cleanup(srv.Close)

	t.Run("authed", func(t *testing.T) {
		c := client.NewClient(srv.URL, testTok, "terraform-provider-circleci/test")
		ctx := context.TODO()
		body := make(map[string]any)
		res, err := c.RequestHelper(ctx, http.MethodGet, "/api/test/hello", nil, &body)
		assert.Assert(t, err)
		assert.Check(t, cmp.Equal(res.StatusCode, http.StatusOK))
		assert.Check(t, cmp.DeepEqual(body, map[string]any{
			"message": "Hello World!",
		}))
	})

	t.Run("no_body", func(t *testing.T) {
		ctx := context.TODO()
		c := client.NewClient(srv.URL, testTok, "terraform-provider-circleci/test")
		res, err := c.RequestHelper(ctx, http.MethodGet, "/api/test/hello", nil, nil)
		assert.Assert(t, err)
		assert.Check(t, cmp.Equal(res.StatusCode, http.StatusOK))
	})

	t.Run("post", func(t *testing.T) {
		ctx := context.TODO()
		c := client.NewClient(srv.URL, testTok, "terraform-provider-circleci/test")
		body := make(map[string]any)
		res, err := c.RequestHelper(ctx, http.MethodPost, "/api/test/echo", map[string]any{
			"foo":  "bar",
			"baz":  "boz",
			"bool": true,
		}, &body)
		assert.Assert(t, err)
		assert.Check(t, cmp.Equal(res.StatusCode, http.StatusOK))
		assert.Check(t, cmp.DeepEqual(body, map[string]any{
			"foo":  "bar",
			"baz":  "boz",
			"bool": true,
		}))
	})

	t.Run("unauthed", func(t *testing.T) {
		ctx := context.TODO()
		c := client.NewClient(srv.URL, "", "terraform-provider-circleci/test")
		body := make(map[string]any)
		res, err := c.RequestHelper(ctx, http.MethodGet, "/api/test/hello", map[string]any{
			"foo": "bar",
		}, &body)
		assert.Check(t, cmp.ErrorContains(err, "401 Unauthorized"))
		assert.Check(t, cmp.ErrorContains(err, "You must log in first."))
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("bad_token", func(t *testing.T) {
		ctx := context.TODO()
		c := client.NewClient(srv.URL, "not-valid", "terraform-provider-circleci/test")
		body := make(map[string]any)
		res, err := c.RequestHelper(ctx, http.MethodGet, "/api/test/hello", map[string]any{
			"foo": "bar",
		}, &body)
		assert.Check(t, cmp.ErrorContains(err, "401 Unauthorized"))
		assert.Check(t, cmp.ErrorContains(err, "Invalid token provided."))
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("429", func(t *testing.T) {
		ctx := context.TODO()
		c := client.NewClient(srv.URL, testTok, "terraform-provider-circleci/test")
		body := make(map[string]any)
		res, err := c.RequestHelper(ctx, http.MethodGet, "/api/test/429", nil, &body)
		assert.Assert(t, err)
		assert.Check(t, cmp.Equal(res.StatusCode, http.StatusOK))
		assert.Check(t, cmp.DeepEqual(body, map[string]any{
			"message": "Successfully retried.",
		}))
	})

	t.Run("500", func(t *testing.T) {
		ctx := context.TODO()
		c := client.NewClient(srv.URL, testTok, "terraform-provider-circleci/test")
		body := make(map[string]any)
		res, err := c.RequestHelper(ctx, http.MethodGet, "/api/test/500", nil, &body)
		assert.Assert(t, err)
		assert.Check(t, cmp.Equal(res.StatusCode, http.StatusOK))
		assert.Check(t, cmp.DeepEqual(body, map[string]any{
			"message": "Successfully retried.",
		}))
	})
}

func TestClient_SetsHeaders(t *testing.T) {
	const (
		testTok = "CCIPAT_e4ba0c0b-3a1e-4f5f-8b0e-1f1b2a3c4d5e"
		testUA  = "terraform-provider-circleci/1.2.3"
	)

	var gotUA, gotToken, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotToken = r.Header.Get("Circle-Token")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, testTok, testUA)
	body := make(map[string]any)
	res, err := c.RequestHelper(context.TODO(), http.MethodGet, "/whatever", nil, &body)
	assert.Assert(t, err)
	assert.Check(t, cmp.Equal(res.StatusCode, http.StatusOK))

	assert.Check(t, cmp.Equal(gotUA, testUA))
	assert.Check(t, cmp.Equal(gotToken, testTok))
	assert.Check(t, cmp.Equal(gotAccept, "application/json"))
}
