// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package runner_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/runner"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const testToken = "CCIPAT_test-runner-token"

func setupTest(t *testing.T) *runner.Service {
	fs := fakecircle.New(testToken)
	err := fs.AddResourceClass("uuid", "some resource_class", "some description")
	assert.Assert(t, err)
	srv := httptest.NewServer(fs)
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, testToken, "terraform-provider-circleci/test")

	return runner.NewServiceWithBaseURL(c, srv.URL)
}

func TestFullRunner(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Create a resource class
	createReq := runner.CreateResourceClassRequest{
		OrganizationID: "some-org",
		ResourceClass:  "test-org/test-resource-class",
		Description:    "Test resource class from SDK",
	}
	resourceClass, err := service.CreateResourceClass(ctx, createReq)
	assert.NilError(t, err)
	assert.Check(t, resourceClass != nil)
	assert.Check(t, resourceClass.Id != "")
	assert.Check(t, resourceClass.ResourceClass == createReq.ResourceClass)
	assert.Check(t, resourceClass.Description == createReq.Description)

	resourceClassID := resourceClass.Id

	// List resource classes
	resourceClasses, err := service.ListResourceClasses(ctx, "test-org", "")
	assert.NilError(t, err)
	assert.Check(t, len(resourceClasses.Items) > 0)

	// Create a token
	createTokenReq := runner.CreateTokenRequest{
		OrganizationID: "someOrg",
		ResourceClass:  createReq.ResourceClass,
		Nickname:       "test-token",
	}
	token, err := service.CreateToken(ctx, createTokenReq)
	assert.NilError(t, err)
	assert.Check(t, token != nil)
	assert.Check(t, token.Id != "")
	assert.Check(t, token.Token != "") // Token value only returned on create
	assert.Check(t, token.Nickname == createTokenReq.Nickname)
	assert.Check(t, token.ResourceClass == createTokenReq.ResourceClass)

	tokenID := token.Id

	// List tokens
	tokens, err := service.ListTokens(ctx, createReq.ResourceClass)
	assert.NilError(t, err)
	assert.Check(t, tokens.Items != nil)
	assert.Check(t, len(tokens.Items) > 0)

	// Delete token
	err = service.DeleteToken(ctx, tokenID)
	assert.NilError(t, err)

	// Try to delete resource class without force (should work now that token is deleted)
	err = service.DeleteResourceClass(ctx, resourceClassID, false)
	assert.NilError(t, err)
}

func TestDeleteResourceClassForce(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Create a resource class
	createReq := runner.CreateResourceClassRequest{
		OrganizationID: "3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
		ResourceClass:  "test-org/test-force-delete",
		Description:    "Test force delete",
	}
	resourceClass, err := service.CreateResourceClass(ctx, createReq)
	assert.NilError(t, err)
	assert.Check(t, resourceClass != nil)

	resourceClassID := resourceClass.Id

	// Create a token
	createTokenReq := runner.CreateTokenRequest{
		OrganizationID: "3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
		ResourceClass:  createReq.ResourceClass,
		Nickname:       "test-token-force",
	}
	token, err := service.CreateToken(ctx, createTokenReq)
	assert.NilError(t, err)
	assert.Check(t, token != nil)

	// Try to delete resource class without force (should fail)
	err = service.DeleteResourceClass(ctx, resourceClassID, false)
	assert.Check(t, err != nil)
	assert.Check(t, cmp.ErrorContains(err, "has tokens"))

	// Delete resource class with force (should succeed even with token)
	err = service.DeleteResourceClass(ctx, resourceClassID, true)
	assert.NilError(t, err)
}

func TestListRunners(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Create a resource class first
	createReq := runner.CreateResourceClassRequest{
		OrganizationID: "3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
		ResourceClass:  "test-org/test-runners-list",
		Description:    "Test runners list",
	}
	resourceClass, err := service.CreateResourceClass(ctx, createReq)
	assert.NilError(t, err)

	// List runners with resource class filter
	params := runner.ListRunnersParams{
		ResourceClass: createReq.ResourceClass,
	}
	runners, err := service.ListRunners(ctx, params)
	assert.NilError(t, err)
	assert.Check(t, runners != nil) // May be empty if no runners connected

	// List runners with namespace filter
	params = runner.ListRunnersParams{
		Namespace: "test-org",
	}
	runners, err = service.ListRunners(ctx, params)
	assert.NilError(t, err)
	assert.Check(t, runners != nil)

	// Clean up
	err = service.DeleteResourceClass(ctx, resourceClass.Id, true)
	assert.NilError(t, err)
}

func TestTaskCounts(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Create a resource class
	createReq := runner.CreateResourceClassRequest{
		OrganizationID: "3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
		ResourceClass:  "test-org/test-task-counts",
		Description:    "Test task counts",
	}
	resourceClass, err := service.CreateResourceClass(ctx, createReq)
	assert.NilError(t, err)

	// Get unclaimed task count
	unclaimedCount, err := service.GetUnclaimedTaskCount(ctx, createReq.ResourceClass)
	assert.NilError(t, err)
	assert.Check(t, unclaimedCount != nil)
	assert.Check(t, unclaimedCount.UnclaimedTaskCount >= 0)

	// Get running task count
	runningCount, err := service.GetRunningTaskCount(ctx, createReq.ResourceClass)
	assert.NilError(t, err)
	assert.Check(t, runningCount != nil)
	assert.Check(t, runningCount.RunningRunnerTasks >= 0)

	// Clean up
	err = service.DeleteResourceClass(ctx, resourceClass.Id, true)
	assert.NilError(t, err)
}

func TestCreateResourceClassDuplicate(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Create a resource class
	createReq := runner.CreateResourceClassRequest{
		OrganizationID: "3ddcf1d1-7f5f-4139-8cef-71ad0921a968",
		ResourceClass:  "test-org/duplicate-test",
		Description:    "First resource class",
	}
	resourceClass, err := service.CreateResourceClass(ctx, createReq)
	assert.NilError(t, err)

	// Try to create duplicate
	_, err = service.CreateResourceClass(ctx, createReq)
	assert.Check(t, err != nil)
	assert.Check(t, cmp.ErrorContains(err, "already exists"))

	// Clean up
	err = service.DeleteResourceClass(ctx, resourceClass.Id, true)
	assert.NilError(t, err)
}

func TestDeleteNonexistentResourceClass(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Try to delete a resource class that doesn't exist
	err := service.DeleteResourceClass(ctx, "nonexistent-id", false)
	assert.Check(t, err != nil)
	assert.Check(t, cmp.ErrorContains(err, "not found"))
}

func TestDeleteNonexistentToken(t *testing.T) {
	ctx := context.TODO()
	service := setupTest(t)

	// Try to delete a token that doesn't exist
	err := service.DeleteToken(ctx, "nonexistent-id")
	assert.Check(t, err != nil)
	assert.Check(t, cmp.ErrorContains(err, "not found"))
}
