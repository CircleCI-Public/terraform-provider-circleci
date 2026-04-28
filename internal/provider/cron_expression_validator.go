// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = cronExpressionValidator{}

type cronExpressionValidator struct{}

func (v cronExpressionValidator) Description(_ context.Context) string {
	return "value must be a valid 5-field cron expression (e.g. \"0 * * * *\")"
}

func (v cronExpressionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cronExpressionValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := validateCronExpression(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Cron Expression", err.Error())
	}
}

// CronExpressionValidator returns a validator that checks for a valid 5-field cron expression.
func CronExpressionValidator() validator.String {
	return cronExpressionValidator{}
}

func validateCronExpression(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cron expression must have exactly 5 fields "+
			"(minute hour day-of-month month day-of-week), got %d", len(fields))
	}

	type fieldSpec struct {
		name string
		min  int
		max  int
	}
	specs := []fieldSpec{
		{"minute", 0, 59},
		{"hour", 0, 23},
		{"day-of-month", 1, 31},
		{"month", 1, 12},
		{"day-of-week", 0, 7},
	}

	for i, spec := range specs {
		if err := validateCronField(fields[i], spec.min, spec.max, spec.name); err != nil {
			return err
		}
	}
	return nil
}

func validateCronField(field string, min, max int, name string) error {
	for _, part := range strings.Split(field, ",") {
		if err := validateCronPart(part, min, max, name); err != nil {
			return err
		}
	}
	return nil
}

func validateCronPart(part string, min, max int, name string) error {
	if part == "*" {
		return nil
	}

	// Handle optional /step suffix (e.g. */5, 1-5/2, 3/10)
	base, stepStr, hasStep := strings.Cut(part, "/")
	if hasStep {
		step, err := strconv.Atoi(stepStr)
		if err != nil || step < 1 {
			return fmt.Errorf("invalid step in %s field %q: step must be a positive integer", name, part)
		}
		if base == "*" {
			return nil
		}
	}

	// base is either a range (n-m) or a single number
	if lo, hi, isRange := strings.Cut(base, "-"); isRange {
		return validateCronRange(lo, hi, min, max, name, base)
	}
	return validateCronNumber(base, min, max, name)
}

func validateCronRange(loStr, hiStr string, min, max int, name, raw string) error {
	lo, err := strconv.Atoi(loStr)
	if err != nil {
		return fmt.Errorf("invalid range in %s field %q: %q is not a number", name, raw, loStr)
	}
	hi, err := strconv.Atoi(hiStr)
	if err != nil {
		return fmt.Errorf("invalid range in %s field %q: %q is not a number", name, raw, hiStr)
	}
	if lo < min || lo > max {
		return fmt.Errorf("range start %d in %s field is out of range [%d, %d]", lo, name, min, max)
	}
	if hi < min || hi > max {
		return fmt.Errorf("range end %d in %s field is out of range [%d, %d]", hi, name, min, max)
	}
	if lo > hi {
		return fmt.Errorf("range start %d must not exceed range end %d in %s field", lo, hi, name)
	}
	return nil
}

func validateCronNumber(s string, min, max int, name string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid value in %s field: %q is not a number", name, s)
	}
	if n < min || n > max {
		return fmt.Errorf("value %d in %s field is out of range [%d, %d]", n, name, min, max)
	}
	return nil
}
