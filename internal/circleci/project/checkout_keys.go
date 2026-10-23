// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package project

import (
	"context"
	"net/http"
	"net/url"

	"terraform-provider-circleci/internal/circleci/common"
)

type CheckoutKey struct {
	PublicKey   string `json:"public_key"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Preferred   bool   `json:"preferred"`
	CreatedAt   string `json:"created_at"`
}

func (s *ProjectService) GetCheckoutKeys(ctx context.Context, slug string) ([]CheckoutKey, error) {
	var keys []CheckoutKey
	var pageToken string
	for {
		path := "/project/" + slug + "/checkout-key"
		if pageToken != "" {
			path += "?page-token=" + url.QueryEscape(pageToken)
		}
		var page common.PaginatedResponse[CheckoutKey]
		_, err := s.client.RequestHelper(ctx, http.MethodGet, path, nil, &page)
		if err != nil {
			return nil, err
		}
		keys = append(keys, page.Items...)
		if page.NextPageToken == "" {
			return keys, nil
		}
		pageToken = page.NextPageToken
	}
}
