// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package connection_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/connection"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	testTok            = "6f0c1a43-2d6b-4b1e-9a51-7c7cf2b1f0a1"
	testOrgID          = "5d7f0b8e-3c64-4b8e-9f5e-0a6f8d2c1b3e"
	connectionAudience = "u:7:0f9e8d7c-6b5a-4c3d-2e1f-0a9b8c7d6e5f"
	checkoutAudience   = "u:7:1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
)

func newService(t *testing.T) *connection.Service {
	t.Helper()
	srv := httptest.NewServer(fakecircle.New(testTok))
	t.Cleanup(srv.Close)
	return connection.NewService(client.NewClient(srv.URL+"/api/v2", testTok, "terraform-provider-circleci/test"))
}

func validSetup() connection.ForgejoSetup {
	return connection.ForgejoSetup{
		URL:                "https://forgejo.example.com/",
		Owner:              "carta",
		ConnectionAudience: connectionAudience,
		CheckoutAudience:   checkoutAudience,
		RequestHeaders:     map[string]string{"x-proxy-token": "s3cret"},
	}
}

func TestService_Forgejo(t *testing.T) {
	ctx := context.TODO()
	svc := newService(t)

	var setup *connection.SetupResult
	t.Run("setup", func(t *testing.T) {
		var err error
		setup, err = svc.SetupForgejo(ctx, testOrgID, validSetup())
		assert.NilError(t, err)
		assert.Check(t, setup.ID != "")
		assert.Check(t, cmp.DeepEqual(setup.Owner, connection.Owner{ID: "42", Login: "carta", Type: "organization"}))
		assert.Check(t, cmp.Equal(setup.Webhook.URL, "https://internal.circleci.com/private/soc/forgejo/webhook/"+setup.ID))
		assert.Check(t, setup.Webhook.Secret != "")
		assert.Check(t, cmp.DeepEqual(setup.Webhook.Events, []string{"push", "pull_request"}))
	})
	assert.Assert(t, setup != nil)

	t.Run("get", func(t *testing.T) {
		conn, err := svc.Get(ctx, setup.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(conn.ID, setup.ID))
		assert.Check(t, cmp.Equal(conn.Provider, "forgejo"))
		assert.Check(t, cmp.Equal(conn.ProviderURL, "https://forgejo.example.com"))
		assert.Check(t, cmp.Equal(conn.ExternalID, "42"))
		assert.Check(t, cmp.Equal(conn.TargetType, "organization"))
		assert.Assert(t, conn.Credentials != nil)
		assert.Check(t, cmp.Equal(conn.Credentials.Access[connection.PurposeConnection].Audience, connectionAudience))
		assert.Check(t, cmp.Equal(conn.Credentials.Access[connection.PurposeCheckout].Audience, checkoutAudience))
		assert.Check(t, cmp.DeepEqual(conn.Credentials.RequestHeaderNames, []string{"X-Proxy-Token"}))
	})

	t.Run("list", func(t *testing.T) {
		conns, err := svc.List(ctx, testOrgID)
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(conns, 1))
		assert.Check(t, cmp.Equal(conns[0].ID, setup.ID))
		assert.Check(t, cmp.Equal(conns[0].Provider, "forgejo"))
	})

	t.Run("a second setup for the org conflicts", func(t *testing.T) {
		_, err := svc.SetupForgejo(ctx, testOrgID, validSetup())
		assert.Check(t, cmp.ErrorContains(err, "409 Conflict"))
	})

	t.Run("set an audience", func(t *testing.T) {
		const newAudience = "u:7:9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a"
		err := svc.SetOIDCAudience(ctx, setup.ID, connection.PurposeCheckout, newAudience)
		assert.NilError(t, err)

		conn, err := svc.Get(ctx, setup.ID)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(conn.Credentials.Access[connection.PurposeCheckout].Audience, newAudience))
	})

	t.Run("clear the request headers", func(t *testing.T) {
		names, err := svc.SetRequestHeaders(ctx, setup.ID, nil)
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(names, 0))
	})

	t.Run("rotate the webhook secret", func(t *testing.T) {
		secret, err := svc.RotateWebhookSecret(ctx, setup.ID)
		assert.NilError(t, err)
		assert.Check(t, secret != "")
		assert.Check(t, secret != setup.Webhook.Secret)

		conn, err := svc.Get(ctx, setup.ID)
		assert.NilError(t, err)
		assert.Check(t, conn.Credentials.WebhookSecret.IsRotationPending)
	})

	t.Run("delete", func(t *testing.T) {
		err := svc.Delete(ctx, setup.ID)
		assert.NilError(t, err)

		_, err = svc.Get(ctx, setup.ID)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "connection should be gone: %v", err)
	})
}

func TestService_SetupForgejoRejected(t *testing.T) {
	ctx := context.TODO()
	svc := newService(t)

	tests := []struct {
		name    string
		modify  func(*connection.ForgejoSetup)
		wantErr string
	}{
		{
			name:    "missing owner",
			modify:  func(s *connection.ForgejoSetup) { s.Owner = "" },
			wantErr: "vcs.owner is required.",
		},
		{
			name:    "missing checkout audience",
			modify:  func(s *connection.ForgejoSetup) { s.CheckoutAudience = "" },
			wantErr: "vcs.checkout_audience is required.",
		},
		{
			name:    "audiences from different users",
			modify:  func(s *connection.ForgejoSetup) { s.CheckoutAudience = "u:8:1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d" },
			wantErr: "must be audiences Forgejo generated for the same user",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setup := validSetup()
			tc.modify(&setup)
			_, err := svc.SetupForgejo(ctx, testOrgID, setup)
			assert.Check(t, cmp.ErrorContains(err, "400 Bad Request"))
			assert.Check(t, cmp.ErrorContains(err, tc.wantErr))
		})
	}
}

func TestService_UnknownConnection(t *testing.T) {
	ctx := context.TODO()
	svc := newService(t)
	const missing = "0e2d4c6b-8a9f-4e1d-b3c5-a7f9e1d3c5b7"

	t.Run("get", func(t *testing.T) {
		_, err := svc.Get(ctx, missing)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("set an audience", func(t *testing.T) {
		err := svc.SetOIDCAudience(ctx, missing, connection.PurposeConnection, connectionAudience)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})

	t.Run("delete", func(t *testing.T) {
		err := svc.Delete(ctx, missing)
		notFound := client.IsNotFound(err)
		assert.Check(t, notFound, "got %v", err)
	})
}
