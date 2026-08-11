// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestTriggerParametersRoundTrip verifies that typed pipeline parameters
// (string, boolean, number) survive the conversion to the SDK's map[string]any
// and back into the dynamic `parameters` attribute — the fix for issue #122.
func TestTriggerParametersRoundTrip(t *testing.T) {
	ctx := context.Background()

	obj, diags := types.ObjectValue(
		map[string]attr.Type{
			"branch":         types.StringType,
			"deploy_enabled": types.BoolType,
			"retries":        types.NumberType,
		},
		map[string]attr.Value{
			"branch":         types.StringValue("main"),
			"deploy_enabled": types.BoolValue(false),
			"retries":        types.NumberValue(big.NewFloat(3)),
		},
	)
	if diags.HasError() {
		t.Fatalf("building object: %v", diags)
	}

	apiParams, diags := triggerParametersToMap(ctx, types.DynamicValue(obj))
	if diags.HasError() {
		t.Fatalf("triggerParametersToMap: %v", diags)
	}

	if got, ok := apiParams["branch"].(string); !ok || got != "main" {
		t.Errorf("branch = %#v, want string %q", apiParams["branch"], "main")
	}
	if got, ok := apiParams["deploy_enabled"].(bool); !ok || got != false {
		t.Errorf("deploy_enabled = %#v, want bool false", apiParams["deploy_enabled"])
	}
	if got, ok := apiParams["retries"].(int64); !ok || got != 3 {
		t.Errorf("retries = %#v, want int64 3", apiParams["retries"])
	}

	// The CircleCI API decodes JSON numbers as float64, so mirror that on the way back.
	back, diags := triggerParametersFromAPI(map[string]any{
		"branch":         "main",
		"deploy_enabled": false,
		"retries":        float64(3),
	})
	if diags.HasError() {
		t.Fatalf("triggerParametersFromAPI: %v", diags)
	}

	backObj, ok := back.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected underlying object, got %T", back.UnderlyingValue())
	}
	if !backObj.Equal(obj) {
		t.Errorf("round-trip mismatch:\n got: %#v\nwant: %#v", backObj, obj)
	}
}

func TestTriggerParametersFromAPIEmptyIsNull(t *testing.T) {
	got, diags := triggerParametersFromAPI(nil)
	if diags.HasError() {
		t.Fatalf("triggerParametersFromAPI: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("expected null dynamic for empty parameters, got %#v", got)
	}
}

func TestTriggerParametersToMapNull(t *testing.T) {
	got, diags := triggerParametersToMap(context.Background(), types.DynamicNull())
	if diags.HasError() {
		t.Fatalf("triggerParametersToMap: %v", diags)
	}
	if got != nil {
		t.Errorf("expected nil map for null parameters, got %#v", got)
	}
}
