// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/pipeline"
)

// forgejoProviderRequiresReplace replaces the resource when a provider
// attribute changes to or from forgejo, because Forgejo pipelines and triggers
// live in the V3 API and cannot be converted to or from another provider.
func forgejoProviderRequiresReplace() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = req.StateValue.ValueString() == pipeline.ProviderForgejo ||
				req.PlanValue.ValueString() == pipeline.ProviderForgejo
		},
		"Changing the provider to or from forgejo forces a new resource.",
		"Changing the provider to or from `forgejo` forces a new resource.",
	)
}

// forgejoRepoRequiresReplace replaces the resource when a configured
// repository attribute changes and providerAttr is forgejo, because the V3 API
// cannot move a Forgejo pipeline or trigger to another repository. An
// unconfigured attribute is computed and never forces replacement.
func forgejoRepoRequiresReplace(providerAttr string) planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			if req.ConfigValue.IsNull() {
				return
			}
			var provider types.String
			resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(providerAttr), &provider)...)
			resp.RequiresReplace = provider.ValueString() == pipeline.ProviderForgejo
		},
		"Changing the repository of a forgejo resource forces a new resource.",
		"Changing the repository of a `forgejo` resource forces a new resource.",
	)
}

// repoUseStateForUnknown keeps a computed repository attribute's prior value
// while the provider and the sibling attribute, the repository's other
// identifier, are unchanged. Otherwise the value is left unknown, because
// CircleCI recomputes it.
func repoUseStateForUnknown(siblingAttr, providerAttr string) planmodifier.String {
	return repoStateModifier{siblingAttr: siblingAttr, providerAttr: providerAttr}
}

type repoStateModifier struct {
	siblingAttr, providerAttr string
}

func (m repoStateModifier) Description(_ context.Context) string {
	return "Keeps the prior value while the repository is unchanged."
}

func (m repoStateModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m repoStateModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || !req.PlanValue.IsUnknown() || req.State.Raw.IsNull() {
		return
	}
	for _, attr := range []string{m.siblingAttr, m.providerAttr} {
		var plan, state types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attr), &plan)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attr), &state)...)
		if resp.Diagnostics.HasError() || plan.IsUnknown() || !plan.Equal(state) {
			return
		}
	}
	resp.PlanValue = req.StateValue
}
