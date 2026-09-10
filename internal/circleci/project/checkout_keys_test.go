// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package project_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/project"
)

func TestProjectService_GetCheckoutKeys(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		status    int
		wantCount int
		wantError string
	}{
		{"paginated", `{"items":[{"public-key":"ssh-ed25519 example","type":"deploy-key","fingerprint":"aa:bb","preferred":true,"created-at":"2026-01-01T00:00:00Z"}],"next_page_token":"a+b/c="}`, 200, 1, ""},
		{"underscore fields", `{"items":[{"public_key":"ssh-ed25519 example","type":"deploy-key","fingerprint":"aa:bb","preferred":true,"created_at":"2026-01-01T00:00:00Z"}]}`, 200, 1, ""},
		{"empty", `{"items":[],"next_page_token":null}`, 200, 0, ""},
		{"forbidden", `{"message":"Forbidden"}`, 403, 0, "403 Forbidden"},
		{"invalid JSON", `{`, 200, 0, "error decoding response body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Check(t, cmp.Equal(r.Method, http.MethodGet))
				assert.Check(t, cmp.Equal(r.URL.Path, "/api/v2/project/bitbucket/org/repo/checkout-key"))
				token := r.Header.Get("Circle-Token")
				assert.Check(t, cmp.Equal(token, testTok))
				if requests > 1 {
					pageToken := r.URL.Query().Get("page-token")
					assert.Check(t, cmp.Equal(pageToken, "a+b/c="))
					fmt.Fprint(w, `{"items":[],"next_page_token":null}`)
					return
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			t.Cleanup(srv.Close)
			s := project.NewProjectService(client.NewClient(srv.URL+"/api/v2", testTok, "test"))
			keys, err := s.GetCheckoutKeys(context.Background(), "bitbucket/org/repo")
			if tc.wantError != "" {
				assert.Check(t, cmp.ErrorContains(err, tc.wantError))
				return
			}
			assert.NilError(t, err)
			assert.Check(t, cmp.Len(keys, tc.wantCount))
			if tc.name == "paginated" {
				assert.Check(t, cmp.Equal(requests, 2))
			}
			if tc.wantCount > 0 {
				assert.Check(t, cmp.DeepEqual(keys, []project.CheckoutKey{{PublicKey: "ssh-ed25519 example", Type: "deploy-key", Fingerprint: "aa:bb", Preferred: true, CreatedAt: "2026-01-01T00:00:00Z"}}))
			}
		})
	}
}
