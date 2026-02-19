// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = webhookURLValidator{}

// webhookURLValidator validates that a webhook URL does not point to local or private addresses.
// This helps prevent SSRF (Server-Side Request Forgery) attacks.
type webhookURLValidator struct{}

// Description describes the validation in plain text formatting.
func (v webhookURLValidator) Description(_ context.Context) string {
	return "URL must not point to localhost, loopback addresses, or private IP ranges"
}

// MarkdownDescription describes the validation in Markdown formatting.
func (v webhookURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v webhookURLValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()

	// Parse the URL
	parsedURL, err := url.Parse(value)
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid URL",
			fmt.Sprintf("Unable to parse URL: %s", err.Error()),
		)
		return
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid URL",
			"URL must contain a hostname",
		)
		return
	}

	// Check for localhost variations
	if isLocalhost(hostname) {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Webhook URL",
			fmt.Sprintf("Webhook URL cannot point to localhost or loopback addresses (got: %s). This is a security risk.", hostname),
		)
		return
	}

	// Try to parse as IP address
	ip := net.ParseIP(hostname)
	if ip != nil {
		// Check if it's a private or reserved IP
		if isPrivateOrReservedIP(ip) {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Invalid Webhook URL",
				fmt.Sprintf("Webhook URL cannot point to private, loopback, or link-local IP addresses (got: %s). This is a security risk.", hostname),
			)
			return
		}
	} else {
		// Not an IP, check for problematic hostnames
		if isProblematicHostname(hostname) {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Invalid Webhook URL",
				fmt.Sprintf("Webhook URL cannot use hostname '%s'. This is a security risk.", hostname),
			)
			return
		}
	}
}

// isLocalhost checks if the hostname is a localhost variation.
func isLocalhost(hostname string) bool {
	hostname = strings.ToLower(hostname)

	// Check for common localhost names
	if hostname == "localhost" ||
		hostname == "localhost.localdomain" ||
		strings.HasSuffix(hostname, ".localhost") {
		return true
	}

	// Check for IPv6 loopback (::1)
	if hostname == "::1" || hostname == "[::1]" {
		return true
	}

	// Check for IPv4 loopback (127.x.x.x)
	ip := net.ParseIP(hostname)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}

// isPrivateOrReservedIP checks if an IP address is private, loopback, or link-local.
func isPrivateOrReservedIP(ip net.IP) bool {
	// Loopback addresses (127.0.0.0/8 for IPv4, ::1 for IPv6)
	if ip.IsLoopback() {
		return true
	}

	// Link-local addresses (169.254.0.0/16 for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for IPv4 addresses
	ipv4 := ip.To4()
	if ipv4 != nil {
		// Private address ranges (RFC 1918)
		// 10.0.0.0/8
		if ipv4[0] == 10 {
			return true
		}
		// 172.16.0.0/12 (172.16.0.0 - 172.31.255.255)
		if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ipv4[0] == 192 && ipv4[1] == 168 {
			return true
		}

		// Other special/reserved ranges
		// 0.0.0.0/8 - Current network
		if ipv4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 - Shared address space (RFC 6598)
		if ipv4[0] == 100 && (ipv4[1]&0xC0) == 64 {
			return true
		}
		// 192.0.0.0/24 - IETF Protocol Assignments
		if ipv4[0] == 192 && ipv4[1] == 0 && ipv4[2] == 0 {
			return true
		}
		// 192.0.2.0/24 - TEST-NET-1
		if ipv4[0] == 192 && ipv4[1] == 0 && ipv4[2] == 2 {
			return true
		}
		// 198.18.0.0/15 - Network interconnect device benchmark testing
		if ipv4[0] == 198 && (ipv4[1] == 18 || ipv4[1] == 19) {
			return true
		}
		// 198.51.100.0/24 - TEST-NET-2
		if ipv4[0] == 198 && ipv4[1] == 51 && ipv4[2] == 100 {
			return true
		}
		// 203.0.113.0/24 - TEST-NET-3
		if ipv4[0] == 203 && ipv4[1] == 0 && ipv4[2] == 113 {
			return true
		}
		// 224.0.0.0/4 - Multicast
		if ipv4[0] >= 224 && ipv4[0] <= 239 {
			return true
		}
		// 240.0.0.0/4 - Reserved for future use
		if ipv4[0] >= 240 {
			return true
		}
	} else {
		// IPv6 addresses
		// Unique local addresses (fc00::/7)
		if len(ip) >= 1 && (ip[0]&0xFE) == 0xFC {
			return true
		}
		// Multicast addresses (ff00::/8)
		if ip.IsMulticast() {
			return true
		}
		// Unspecified address (::)
		if ip.IsUnspecified() {
			return true
		}
	}

	return false
}

// isProblematicHostname checks for hostnames that could be problematic.
func isProblematicHostname(hostname string) bool {
	hostname = strings.ToLower(hostname)

	// Check for .local TLD (used for mDNS/Bonjour)
	if strings.HasSuffix(hostname, ".local") {
		return true
	}

	// Check for .internal TLD (commonly used for internal networks)
	if strings.HasSuffix(hostname, ".internal") {
		return true
	}

	return false
}

// WebhookURLValidator returns a validator that ensures webhook URLs are not pointing
// to local or private addresses, helping prevent SSRF attacks.
func WebhookURLValidator() validator.String {
	return webhookURLValidator{}
}
