// Copyright (c) Circle Internet Services, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/group"
)

const (
	fakeGroupUserID     = "2a4c6e8a-0b1d-4f3a-9c5e-7a9b1c3d5e7f"
	groupAddress        = "circleci_group.platform"
	groupMemberAddress  = "circleci_group_member.alice"
	groupProjectAddress = "circleci_group_project_role.service"
)

func testAccGroupConfig(host, orgID, name, role string) string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %[1]q
  key  = %[2]q
}

resource "circleci_project" "service" {
  name            = "service"
  organization_id = %[3]q
}

resource "circleci_group" "platform" {
  organization_id = %[3]q
  name            = %[4]q
}

resource "circleci_group_member" "alice" {
  organization_id = %[3]q
  group_id        = circleci_group.platform.id
  user_id         = %[5]q
}

resource "circleci_group_project_role" "service" {
  organization_id = %[3]q
  group_id        = circleci_group.platform.id
  project_id      = circleci_project.service.id
  role            = %[6]q
}
`, host, fakeForgejoToken, orgID, name, fakeGroupUserID, role)
}

func importID(address string, attrs ...string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return "", fmt.Errorf("%s not found in state", address)
		}
		id := ""
		for i, attr := range attrs {
			if i > 0 {
				id += "/"
			}
			id += rs.Primary.Attributes[attr]
		}
		return id, nil
	}
}

func TestAccGroupResources(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(f.host, orgID, "platform", "project-viewer"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(groupAddress, tfjsonpath.New("description"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(groupProjectAddress, tfjsonpath.New("role"), knownvalue.StringExact("project-viewer")),
				},
			},
			{
				Config: testAccGroupConfig(f.host, orgID, "platform-eng", "project-contributor"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(groupAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction(groupMemberAddress, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(groupProjectAddress, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(groupAddress, tfjsonpath.New("name"), knownvalue.StringExact("platform-eng")),
					statecheck.ExpectKnownValue(groupProjectAddress, tfjsonpath.New("role"), knownvalue.StringExact("project-contributor")),
				},
			},
			{
				ResourceName:      groupAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(groupAddress, "organization_id", "id"),
			},
			{
				ResourceName:      groupMemberAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(groupMemberAddress, "organization_id", "group_id", "user_id"),
			},
			{
				ResourceName:      groupProjectAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(groupProjectAddress, "organization_id", "group_id", "project_id"),
			},
		},
	})
}

func TestAccGroupMemberRemovedOutOfBand(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)
	config := testAccGroupConfig(f.host, orgID, "platform", "project-viewer")
	groups := group.NewService(client.NewClient(f.host, fakeForgejoToken, "terraform-provider-circleci/test"))

	var groupID string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					groupID = s.RootModule().Resources[groupAddress].Primary.ID
					return nil
				},
			},
			{
				PreConfig: func() {
					err := groups.RemoveMembers(context.TODO(), orgID, groupID, fakeGroupUserID)
					assert.NilError(t, err)
					projectID := f.fake.ProjectIDByName("service")
					err = groups.RevokeProjects(context.TODO(), orgID, groupID, projectID)
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(groupMemberAddress, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction(groupProjectAddress, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccGroupProjectRoleInvalid(t *testing.T) {
	f, orgID := newFakeForgejoOrg(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGroupConfig(f.host, orgID, "platform", "project-owner"),
				ExpectError: regexp.MustCompile(`Attribute role value must be one of`),
			},
		},
	})
}
