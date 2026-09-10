// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"terraform-provider-circleci/internal/circleci/common"
)

type CheckoutKey struct {
	PublicKey   string `json:"public-key"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Preferred   bool   `json:"preferred"`
	CreatedAt   string `json:"created-at"`
}

func (k *CheckoutKey) UnmarshalJSON(data []byte) error {
	type checkoutKey CheckoutKey
	var decoded struct {
		checkoutKey
		PublicKey *string `json:"public_key"`
		CreatedAt *string `json:"created_at"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*k = CheckoutKey(decoded.checkoutKey)
	if decoded.PublicKey != nil {
		k.PublicKey = *decoded.PublicKey
	}
	if decoded.CreatedAt != nil {
		k.CreatedAt = *decoded.CreatedAt
	}
	return nil
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
