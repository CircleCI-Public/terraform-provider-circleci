package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type webhookEventNameValidator struct{}

// Description provides the user-friendly description for the error.
func (v webhookEventNameValidator) Description(_ context.Context) string {
	return "must be set when event_source_provider is 'webhook'"
}

// MarkdownDescription is the detailed markdown for documentation.
func (v webhookEventNameValidator) MarkdownDescription(_ context.Context) string {
	return "Requires 'event_name' if 'event_source_provider' is 'webhook'."
}

// Validate checks the condition.
func (v webhookEventNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// 1. Get the value of event_source_provider from the configuration
	var providerValue types.String

	// req.Config can retrieve sibling attributes by path
	req.Config.GetAttribute(ctx, path.Root("event_source_provider"), &providerValue)

	// Check if we are in the "webhook" case
	if providerValue.ValueString() == "webhook" {
		// If event_source_provider is webhook, event_name cannot be null or empty
		if req.ConfigValue.IsNull() || req.ConfigValue.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Missing required attribute: event_name",
				"The 'event_name' attribute is required when 'event_source_provider' is set to 'webhook'.",
			)
		}
	}
}

// NewWebhookEventNameValidator is a helper function to instantiate the validator.
func NewWebhookEventNameValidator() validator.String {
	return webhookEventNameValidator{}
}
