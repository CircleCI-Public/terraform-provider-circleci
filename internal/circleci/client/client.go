// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"

	"terraform-provider-circleci/internal/circleci/closer"
)

type Client struct {
	baseURL   string
	client    *retryablehttp.Client
	authToken string
	userAgent string
}

func NewClient(baseURL, authToken, userAgent string) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10

	return &Client{
		baseURL:   baseURL,
		client:    retryClient,
		authToken: authToken,
		userAgent: userAgent,
	}
}

func (c *Client) request(ctx context.Context, url, method string, body, respBody any) (*Response, error) {
	return c.do(ctx, url, method, body, respBody, func(h http.Header) {
		h.Set("Circle-Token", c.authToken)
	})
}

func (c *Client) do(
	ctx context.Context, url, method string, body, respBody any, setAuth func(http.Header),
) (_ *Response, err error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	req, err := retryablehttp.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	setAuth(req.Header)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer closer.ErrorHandler(res.Body, &err)
	defer func() {
		// This helps with connection pooling (makes sure there's nothing trailing in the HTTP request like newlines)
		_, _ = io.Copy(io.Discard, res.Body)
	}()

	if res.StatusCode >= 400 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("error Reading response: %w", err)
		}
		return nil, &APIError{StatusCode: res.StatusCode, Status: res.Status, Body: string(body)}
	}

	if respBody != nil {
		b, err := io.ReadAll(res.Body) // Body is now empty
		if err != nil {
			return nil, fmt.Errorf("error reading body: %s: %w", string(b), err)
		}

		// Pass the byte slice to Unmarshal instead of the Reader to Decoder
		if err := json.Unmarshal(b, respBody); err != nil {
			return nil, fmt.Errorf("error decoding response body: %s: %w", string(b), err)
		}
	}

	return &Response{
		StatusCode: res.StatusCode,
	}, nil
}

type Response struct {
	StatusCode int
}

// APIError is returned for any response with a 4xx or 5xx status, so callers
// can branch on the status code instead of matching on the message.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.Body)
}

// HTTPStatusCode reports the response status code.
func (e *APIError) HTTPStatusCode() int {
	return e.StatusCode
}

// IsNotFound reports whether err is an APIError for a 404 response.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// RequestHelperAbsolute is the same as RequestHelper but allows to do a request to other APIs.
func (c *Client) RequestHelperAbsolute(ctx context.Context, method, path string, body, respBody any) (*Response, error) {
	return c.request(ctx, path, method, body, respBody)
}

func (c *Client) RequestHelper(ctx context.Context, method, path string, reqBody, respBody any) (*Response, error) {
	return c.request(ctx, c.baseURL+path, method, reqBody, respBody)
}

// VersionedURL builds an absolute URL for a CircleCI API version other than the
// one in baseURL. baseURL is typically https://host/api/v2; version "v1.1" and
// path "/project/gh/org/repo/settings" become https://host/api/v1.1/project/gh/org/repo/settings.
func (c *Client) VersionedURL(version, path string) string {
	base := strings.TrimSuffix(c.baseURL, "/")
	if idx := strings.LastIndex(base, "/api/"); idx >= 0 {
		base = base[:idx]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + "/api/" + version + path
}
