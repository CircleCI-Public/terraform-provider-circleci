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

func TestCronExpressionValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		expr        string
		expectError bool
	}{
		// Valid expressions
		{name: "every minute", expr: "* * * * *", expectError: false},
		{name: "top of every hour", expr: "0 * * * *", expectError: false},
		{name: "every 5 minutes", expr: "*/5 * * * *", expectError: false},
		{name: "specific time", expr: "30 14 * * *", expectError: false},
		{name: "range in hour", expr: "0 9-17 * * *", expectError: false},
		{name: "range with step", expr: "0 0-23/2 * * *", expectError: false},
		{name: "list in minute", expr: "0,15,30,45 * * * *", expectError: false},
		{name: "specific day of week", expr: "0 9 * * 1", expectError: false},
		{name: "day of week 7 (Sunday)", expr: "0 0 * * 7", expectError: false},
		{name: "specific month", expr: "0 0 1 12 *", expectError: false},
		{name: "fully qualified", expr: "5 4 1 1 0", expectError: false},
		{name: "step on day of month", expr: "0 0 */10 * *", expectError: false},
		{name: "combined list and range", expr: "0 8,12,17 * * 1-5", expectError: false},
		{name: "step from base", expr: "0 6/6 * * *", expectError: false},

		// Wrong field count
		{name: "too few fields", expr: "* * * *", expectError: true},
		{name: "too many fields", expr: "* * * * * *", expectError: true},
		{name: "empty string", expr: "", expectError: true},

		// Minute out of range
		{name: "minute 60", expr: "60 * * * *", expectError: true},
		{name: "minute -1", expr: "-1 * * * *", expectError: true},

		// Hour out of range
		{name: "hour 24", expr: "0 24 * * *", expectError: true},

		// Day of month out of range
		{name: "day 0", expr: "0 0 0 * *", expectError: true},
		{name: "day 32", expr: "0 0 32 * *", expectError: true},

		// Month out of range
		{name: "month 0", expr: "0 0 1 0 *", expectError: true},
		{name: "month 13", expr: "0 0 1 13 *", expectError: true},

		// Day of week out of range
		{name: "dow 8", expr: "0 0 * * 8", expectError: true},

		// Invalid range
		{name: "inverted range", expr: "0 10-5 * * *", expectError: true},
		{name: "non-numeric range start", expr: "0 a-5 * * *", expectError: true},

		// Invalid step
		{name: "zero step", expr: "*/0 * * * *", expectError: true},
		{name: "non-numeric step", expr: "*/a * * * *", expectError: true},

		// Non-numeric value
		{name: "non-numeric minute", expr: "x * * * *", expectError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: types.StringValue(tc.expr),
			}
			resp := &validator.StringResponse{}
			CronExpressionValidator().ValidateString(context.Background(), req, resp)

			if tc.expectError && !resp.Diagnostics.HasError() {
				t.Errorf("expected validation error for %q but got none", tc.expr)
			}
			if !tc.expectError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected validation error for %q: %s", tc.expr, resp.Diagnostics)
			}
		})
	}
}

func TestCronExpressionValidatorSkipsNullAndUnknown(t *testing.T) {
	t.Parallel()

	for _, val := range []types.String{types.StringNull(), types.StringUnknown()} {
		req := validator.StringRequest{Path: path.Root("test"), ConfigValue: val}
		resp := &validator.StringResponse{}
		CronExpressionValidator().ValidateString(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("expected no error for %v, got: %s", val, resp.Diagnostics)
		}
	}
}
