package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ignoreComputedIfGithubAppModifier struct{}

// NewIgnoreComputedIfGithubAppModifier is a helper function.
func NewIgnoreComputedIfGithubAppModifier() planmodifier.String {
	return ignoreComputedIfGithubAppModifier{}
}

// Implementation of the Description method (Required).
func (m ignoreComputedIfGithubAppModifier) Description(ctx context.Context) string {
	return "Ignores the event_name attribute if event_source_provider is 'github_app' and event_name is unconfigured."
}

// Implementation of the MarkdownDescription method (Required).
func (m ignoreComputedIfGithubAppModifier) MarkdownDescription(ctx context.Context) string {
	return "Ignores the event_name attribute if event_source_provider is 'github_app' and event_name is unconfigured. This prevents plan drift when the API returns a default value."
}

func (m ignoreComputedIfGithubAppModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// 1. If the user explicitly set a value, we stop and let the plan use it.
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}

	var providerValue types.String
	// Retrieve event_source_provider from the plan
	req.Plan.GetAttribute(ctx, path.Root("event_source_provider"), &providerValue)

	// 2. Check the condition: only apply logic for 'github_app'
	if providerValue.ValueString() == "github_app" {

		// If the configuration is null (user omitted it) AND we are in the
		// github_app case, we force the planned value to be null.
		// This ensures the attribute is considered unset, preventing the
		// provider's Read function from causing a plan drift.
		resp.PlanValue = types.StringNull()
	}
	// If we're not github_app, we allow the regular flow (which will correctly
	// require event_name for 'webhook' via your validator).
}
