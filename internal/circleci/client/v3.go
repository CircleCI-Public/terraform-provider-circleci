// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/url"
)

// V3 sends a request to the CircleCI V3 API. path is relative to /api/v3 and
// query may be nil. V3 only accepts the token as a bearer token, so this does
// not send the Circle-Token header the other API versions use.
func (c *Client) V3(
	ctx context.Context, method, path string, query url.Values, reqBody, respBody any,
) (*Response, error) {
	u := c.VersionedURL("v3", path)
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return c.do(ctx, u, method, reqBody, respBody, func(h http.Header) {
		h.Set("Authorization", "Bearer "+c.authToken)
	})
}

// V3Ref points at another V3 entity by its ID.
type V3Ref struct {
	ID string `json:"id"`
}

// V3Entity is the shape V3 uses for a single entity, in requests and responses.
type V3Entity[A, R any] struct {
	ID         string `json:"id,omitempty"`
	Attributes A      `json:"attributes"`
	References R      `json:"references,omitzero"`
}

// V3Data wraps a single entity, as every V3 single-entity body does.
type V3Data[T any] struct {
	Data T `json:"data"`
}

// V3Page holds the opaque cursors of a V3 collection page.
type V3Page struct {
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

// V3Collection is one page of a V3 collection.
type V3Collection[T any] struct {
	Data []T    `json:"data"`
	Page V3Page `json:"page"`
}

// V3ListAll fetches every page of a V3 collection, following page.next.
func V3ListAll[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	var all []T
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	for {
		var page V3Collection[T]
		if _, err := c.V3(ctx, http.MethodGet, path, q, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.Page.Next == nil || *page.Page.Next == "" {
			return all, nil
		}
		q.Set("page[cursor]", *page.Page.Next)
	}
}
