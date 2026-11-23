// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// requiresReplaceIfPreviouslyKnown replaces the resource when a value it
// already held changes. An attribute Read cannot fill in, because CircleCI does
// not report it in the configured form, is null after an import, and taking it
// from the configuration must not replace the resource.
func requiresReplaceIfPreviouslyKnown() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull()
		},
		"Changing this value forces a new resource, unless it was unknown since an import.",
		"Changing this value forces a new resource, unless it was unknown since an import.",
	)
}
