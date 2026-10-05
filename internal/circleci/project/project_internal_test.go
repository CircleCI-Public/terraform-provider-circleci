// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package project

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// TestOSSNotSettable covers how CircleCI's refusal to write the oss flag is
// recognised. The fake answers with the message the live API sends today, so
// this is where the wording drifting is accounted for.
func TestOSSNotSettable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no error",
		},
		{
			name: "the refusal CircleCI sends today",
			err:  errors.New(`422 Unprocessable Entity: {"message":"Feature flag 'oss' is not settable for this project."}`),
			want: true,
		},
		{
			name: "a reworded refusal",
			err:  errors.New(`422 Unprocessable Content: {"message":"Open source builds cannot be enabled for this project."}`),
			want: true,
		},
		{
			name: "the message under another status",
			err:  errors.New(`403 Forbidden: {"message":"Feature flag 'oss' is not settable for this project."}`),
			want: true,
		},
		{
			name: "an unrelated failure",
			err:  errors.New(`404 Not Found: {"message":"Project not found"}`),
		},
		{
			name: "a body that happens to mention 422",
			err:  errors.New(`500 Internal Server Error: {"message":"upstream answered 422"}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ossNotSettable(tc.err)
			assert.Check(t, cmp.Equal(got, tc.want))
		})
	}
}
