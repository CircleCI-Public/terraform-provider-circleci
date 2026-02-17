// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWebhookURLValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		// Valid URLs
		{
			name:        "valid public domain",
			url:         "https://example.com/webhook",
			expectError: false,
		},
		{
			name:        "valid public subdomain",
			url:         "https://api.example.com/webhook",
			expectError: false,
		},
		{
			name:        "valid public IP",
			url:         "https://8.8.8.8/webhook",
			expectError: false,
		},
		{
			name:        "valid with port",
			url:         "https://example.com:8443/webhook",
			expectError: false,
		},

		// Localhost variations - should be blocked
		{
			name:        "localhost",
			url:         "https://localhost/webhook",
			expectError: true,
		},
		{
			name:        "localhost with port",
			url:         "https://localhost:8080/webhook",
			expectError: true,
		},
		{
			name:        "localhost.localdomain",
			url:         "https://localhost.localdomain/webhook",
			expectError: true,
		},
		{
			name:        "subdomain.localhost",
			url:         "https://api.localhost/webhook",
			expectError: true,
		},

		// IPv4 loopback - should be blocked
		{
			name:        "127.0.0.1",
			url:         "https://127.0.0.1/webhook",
			expectError: true,
		},
		{
			name:        "127.0.0.1 with port",
			url:         "https://127.0.0.1:8080/webhook",
			expectError: true,
		},
		{
			name:        "127.1.1.1",
			url:         "https://127.1.1.1/webhook",
			expectError: true,
		},

		// IPv6 loopback - should be blocked
		{
			name:        "::1",
			url:         "https://[::1]/webhook",
			expectError: true,
		},

		// Private IPv4 ranges - should be blocked
		{
			name:        "10.0.0.1 (Class A private)",
			url:         "https://10.0.0.1/webhook",
			expectError: true,
		},
		{
			name:        "10.255.255.254 (Class A private)",
			url:         "https://10.255.255.254/webhook",
			expectError: true,
		},
		{
			name:        "172.16.0.1 (Class B private)",
			url:         "https://172.16.0.1/webhook",
			expectError: true,
		},
		{
			name:        "172.31.255.254 (Class B private)",
			url:         "https://172.31.255.254/webhook",
			expectError: true,
		},
		{
			name:        "192.168.0.1 (Class C private)",
			url:         "https://192.168.0.1/webhook",
			expectError: true,
		},
		{
			name:        "192.168.1.1 (Class C private)",
			url:         "https://192.168.1.1/webhook",
			expectError: true,
		},
		{
			name:        "192.168.255.254 (Class C private)",
			url:         "https://192.168.255.254/webhook",
			expectError: true,
		},

		// Link-local - should be blocked
		{
			name:        "169.254.1.1 (Link-local)",
			url:         "https://169.254.1.1/webhook",
			expectError: true,
		},

		// Test networks - should be blocked
		{
			name:        "192.0.2.1 (TEST-NET-1)",
			url:         "https://192.0.2.1/webhook",
			expectError: true,
		},
		{
			name:        "198.51.100.1 (TEST-NET-2)",
			url:         "https://198.51.100.1/webhook",
			expectError: true,
		},
		{
			name:        "203.0.113.1 (TEST-NET-3)",
			url:         "https://203.0.113.1/webhook",
			expectError: true,
		},

		// Shared address space - should be blocked
		{
			name:        "100.64.0.1 (Shared address space)",
			url:         "https://100.64.0.1/webhook",
			expectError: true,
		},

		// Multicast - should be blocked
		{
			name:        "224.0.0.1 (Multicast)",
			url:         "https://224.0.0.1/webhook",
			expectError: true,
		},

		// Reserved hostnames - should be blocked
		{
			name:        ".local TLD",
			url:         "https://server.local/webhook",
			expectError: true,
		},
		{
			name:        ".internal TLD",
			url:         "https://server.internal/webhook",
			expectError: true,
		},

		// Edge cases that should be allowed
		{
			name:        "1.1.1.1 (Cloudflare DNS)",
			url:         "https://1.1.1.1/webhook",
			expectError: false,
		},
		{
			name:        "public IP in private-looking range",
			url:         "https://172.15.0.1/webhook",
			expectError: false,
		},
		{
			name:        "192.169.1.1 (not in 192.168.x.x)",
			url:         "https://192.169.1.1/webhook",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tt.url),
			}
			response := validator.StringResponse{}

			v := WebhookURLValidator()
			v.ValidateString(context.Background(), request, &response)

			hasError := response.Diagnostics.HasError()
			if hasError != tt.expectError {
				if tt.expectError {
					t.Errorf("Expected validation error for URL %q, but got none", tt.url)
				} else {
					t.Errorf("Expected no validation error for URL %q, but got: %v", tt.url, response.Diagnostics.Errors())
				}
			}
		})
	}
}

func TestWebhookURLValidator_NullAndUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value types.String
	}{
		{
			name:  "null value",
			value: types.StringNull(),
		},
		{
			name:  "unknown value",
			value: types.StringUnknown(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: tt.value,
			}
			response := validator.StringResponse{}

			v := WebhookURLValidator()
			v.ValidateString(context.Background(), request, &response)

			if response.Diagnostics.HasError() {
				t.Errorf("Expected no validation error for %s, but got: %v", tt.name, response.Diagnostics.Errors())
			}
		})
	}
}
