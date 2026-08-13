// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/runner"
	"terraform-provider-circleci/internal/circleci/trigger"
)

// The acceptance tests all share one CircleCI org, and the terraform-version CI
// jobs run concurrently, so anything a test creates has to be unique per run.
// Most resources take a random name; the ones whose name *is* their identity get
// it from testAccUniqueName, which also embeds the creation time so that a
// leftover from a run that died before its destroy step can be told apart from a
// concurrent run's live resource.
const (
	// testAccFixturePrefix marks a name as created by these tests.
	testAccFixturePrefix = "acc-test-"

	// testAccStaleAfter is how old a leftover must be before a cleanup deletes
	// it. A job takes ~90s, so this is a wide margin over the longest plausible
	// run while still reclaiming orphans the same day.
	testAccStaleAfter = 15 * time.Minute
)

// testAccUniqueName returns a name of the form acc-test-<purpose>-<unix>-<rand>.
//
// The timestamp lives in the name because the runner resource class API returns
// no creation time — only id, resource_class and description — so the name is
// the only place a cleanup can read an age from. Deleting purely on the prefix
// would delete a concurrent job's resource class.
func testAccUniqueName(purpose string) string {
	return fmt.Sprintf("%s%s-%d-%s",
		testAccFixturePrefix, purpose, time.Now().Unix(), strings.ToLower(rand.Text()[:8]))
}

// The runner fixtures all live in one org, under one namespace.
const (
	testAccRunnerNamespace = "cci-terraform-test"
	testAccRunnerOrgID     = "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
)

// testAccRunnerResourceClass returns a unique, namespaced resource class name.
func testAccRunnerResourceClass(purpose string) string {
	return testAccRunnerNamespace + "/" + testAccUniqueName(purpose)
}

// testAccNameAge reports how long ago a name built by testAccUniqueName was
// created. It returns false for a name it cannot parse, so anything unrecognised
// is never treated as stale — hand-made fixtures are left alone.
func testAccNameAge(name string) (time.Duration, bool) {
	if !strings.Contains(name, testAccFixturePrefix) {
		return 0, false
	}

	// <purpose> may itself contain hyphens, so read from the end: the timestamp
	// is the second-to-last segment and the random tail is the last.
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return 0, false
	}

	secs, err := strconv.ParseInt(parts[len(parts)-2], 10, 64)
	if err != nil {
		return 0, false
	}

	return time.Since(time.Unix(secs, 0)), true
}

// testAccClient builds an API client from the same environment the provider
// reads, for the housekeeping these tests do outside of terraform.
func testAccClient(t *testing.T) *client.Client {
	t.Helper()

	host := os.Getenv("CIRCLE_HOST")
	if host == "" {
		host = "https://circleci.com/api/v2"
	}

	return client.NewClient(host, os.Getenv("CIRCLE_TOKEN"), "terraform-provider-circleci/test")
}

// testAccCleanupStaleTriggers deletes triggers matching an event source that a
// previous run left on a pipeline definition.
//
// The repo-backed trigger tests share fixed pipeline definitions. Identical
// triggers are allowed to coexist, so a leftover is harmless to the test itself,
// but they would otherwise accumulate one per crashed run. Only triggers older
// than testAccStaleAfter are removed, which is what keeps this from deleting a
// concurrent job's trigger. Matching on provider and repo rather than deleting
// every trigger keeps it away from the unrelated fixtures on these definitions.
//
// Cleanup is best effort: a failure here is logged, not fatal, since the test
// can create its trigger regardless.
func testAccCleanupStaleTriggers(t *testing.T, projectID, pipelineID, provider, repoExternalID string) {
	t.Helper()

	svc := trigger.NewTriggerService(testAccClient(t))
	ctx := context.Background()

	existing, err := svc.List(ctx, projectID, pipelineID)
	if err != nil {
		t.Logf("cleanup: listing triggers on pipeline definition %s: %v", pipelineID, err)
		return
	}

	for _, tr := range existing {
		if tr.EventSource.Provider != provider || tr.EventSource.Repo.ExternalId != repoExternalID {
			continue
		}

		created, err := time.Parse(time.RFC3339, tr.CreatedAt)
		if err != nil {
			t.Logf("cleanup: skipping trigger %s, unparseable created_at %q", tr.ID, tr.CreatedAt)
			continue
		}

		if age := time.Since(created); age < testAccStaleAfter {
			t.Logf("cleanup: leaving trigger %s, only %s old", tr.ID, age.Round(time.Second))
			continue
		}

		if err := svc.Delete(ctx, projectID, tr.ID); err != nil {
			t.Logf("cleanup: deleting stale %s trigger %s: %v", provider, tr.ID, err)
			continue
		}
		t.Logf("cleanup: deleted stale %s trigger %s (created %s)", provider, tr.ID, tr.CreatedAt)
	}
}

// testAccCleanupStaleResourceClasses deletes runner resource classes that a
// previous run left in the namespace. Their age comes from the name, since the
// API reports no creation time. Force-delete is used so a class that still has
// tokens is removed rather than skipped.
//
// Cleanup is best effort: a failure here is logged, not fatal.
func testAccCleanupStaleResourceClasses(t *testing.T) {
	t.Helper()

	svc := runner.NewService(testAccClient(t))
	if host := os.Getenv("CIRCLE_RUNNER_HOST"); host != "" {
		svc = runner.NewServiceWithBaseURL(testAccClient(t), host)
	}

	ctx := context.Background()

	classes, err := svc.ListResourceClasses(ctx, testAccRunnerNamespace, testAccRunnerOrgID)
	if err != nil {
		t.Logf("cleanup: listing resource classes in %s: %v", testAccRunnerNamespace, err)
		return
	}

	for _, rc := range classes.Items {
		age, ok := testAccNameAge(rc.ResourceClass)
		if !ok {
			continue // not ours, or hand-made — leave it
		}

		if age < testAccStaleAfter {
			t.Logf("cleanup: leaving resource class %s, only %s old", rc.ResourceClass, age.Round(time.Second))
			continue
		}

		if err := svc.DeleteResourceClass(ctx, rc.Id, true); err != nil {
			t.Logf("cleanup: deleting stale resource class %s: %v", rc.ResourceClass, err)
			continue
		}
		t.Logf("cleanup: deleted stale resource class %s (%s old)", rc.ResourceClass, age.Round(time.Second))
	}
}
