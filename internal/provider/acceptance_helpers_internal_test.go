// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestAccNameAgeOnlyMatchesOurNames guards the predicate that decides whether a
// cleanup deletes a resource. Anything it cannot parse must be reported as not
// ours, so a hand-made fixture is never destroyed.
func TestAccNameAgeOnlyMatchesOurNames(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		wantOur bool
	}{
		{"generated name", testAccUniqueName("runner"), true},
		{"generated name with hyphenated purpose", testAccUniqueName("runner-force"), true},
		{"namespaced generated name", testAccRunnerResourceClass("runner-ds"), true},

		// Everything below must be left alone.
		{"hand-made fixture", "cci-terraform-test/synthetics", false},
		{"the not-found probe", "cci-terraform-test/does-not-exist-acc", false},
		{"namespace alone", "cci-terraform-test", false},
		{"prefix but no timestamp", "acc-test-runner", false},
		{"prefix with non-numeric timestamp", "acc-test-runner-abc-xyz", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := testAccNameAge(tc.input)
			if ok != tc.wantOur {
				t.Errorf("testAccNameAge(%q) recognised=%v, want %v", tc.input, ok, tc.wantOur)
			}
		})
	}
}

// TestAccNameAgeReadsTheEmbeddedTimestamp checks the age itself, since it is what
// separates a leftover from a concurrent run's live resource.
func TestAccNameAgeReadsTheEmbeddedTimestamp(t *testing.T) {
	fresh := testAccUniqueName("runner")
	age, ok := testAccNameAge(fresh)
	if !ok {
		t.Fatalf("testAccNameAge(%q) did not recognise a freshly generated name", fresh)
	}
	if age > time.Minute {
		t.Errorf("a freshly generated name reported an age of %s, want under a minute", age)
	}
	if age >= testAccStaleAfter {
		t.Errorf("a freshly generated name is already considered stale (%s >= %s)", age, testAccStaleAfter)
	}

	old := fmt.Sprintf("%srunner-%d-abcd1234",
		testAccFixturePrefix, time.Now().Add(-2*testAccStaleAfter).Unix())
	age, ok = testAccNameAge(old)
	if !ok {
		t.Fatalf("testAccNameAge(%q) did not recognise a back-dated name", old)
	}
	if age < testAccStaleAfter {
		t.Errorf("a name back-dated by 2x the window reported %s, want at least %s", age, testAccStaleAfter)
	}
}

// TestAccUniqueNameIsUnique guards the other half of the contract: two concurrent
// jobs must not generate the same name within the same second.
func TestAccUniqueNameIsUnique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for range 100 {
		n := testAccUniqueName("runner")
		if seen[n] {
			t.Fatalf("testAccUniqueName produced a duplicate: %q", n)
		}
		seen[n] = true

		if !strings.HasPrefix(n, testAccFixturePrefix) {
			t.Fatalf("testAccUniqueName(%q) lacks the %q marker", n, testAccFixturePrefix)
		}
	}
}
