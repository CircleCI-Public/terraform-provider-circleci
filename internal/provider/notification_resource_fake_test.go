// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/notification"
	"terraform-provider-circleci/internal/circleci/testing/fakecircle"
)

const (
	fakeNotificationToken = "fake-notification-token"
	slackAddress          = "circleci_project_slack_notification.builds"
	preferencesAddress    = "circleci_project_notification_preferences.builds"
	preferencesDataSource = "data.circleci_notification_preferences.builds"

	prefDefaultBranch = "c9930e26-d1e1-46e2-a5d2-45c77843fcd2"
	prefApproval      = "3b5d94e8-6feb-4eb6-8458-ae2571fb4192"
	prefSucceeded     = "796a37f9-7b11-4e92-9322-927e76583cf3"
	prefFailed        = "4234f1ce-9bcd-4a9c-966c-4e365c69e1ed"
	prefInconclusive  = "74d43e43-955f-4596-bda6-c0ef96ba3119"
)

type fakeNotifications struct {
	fake      *fakecircle.Service
	host      string
	orgID     string
	projectID string
}

// newFakeNotifications seeds an organization with one project in the fake.
func newFakeNotifications(t *testing.T) fakeNotifications {
	t.Helper()
	fake := fakecircle.New(fakeNotificationToken)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	org, err := fake.AddOrg(fakecircle.NewOrg{Type: fakecircle.TypeCircleCI, Name: "carta"})
	assert.NilError(t, err)
	p, err := fake.AddProject(fakecircle.NewProject{OrgID: org.ID, Name: "service"})
	assert.NilError(t, err)

	return fakeNotifications{
		fake:      fake,
		host:      srv.URL + "/api/v2",
		orgID:     org.ID.String(),
		projectID: p.ID.String(),
	}
}

func (f fakeNotifications) service() *notification.Service {
	return notification.NewService(client.NewClient(f.host, fakeNotificationToken, "terraform-provider-circleci/test"))
}

func (f fakeNotifications) providerConfig() string {
	return fmt.Sprintf(`
provider "circleci" {
  host = %q
  key  = %q
}
`, f.host, fakeNotificationToken)
}

func (f fakeNotifications) slackConfig(channel, enabled string) string {
	enabledLine := ""
	if enabled != "" {
		enabledLine = "enabled = " + enabled
	}
	return f.providerConfig() + fmt.Sprintf(`
resource "circleci_project_slack_notification" "builds" {
  organization_id = %q
  project_id      = %q
  channel         = %q
  %s
}
`, f.orgID, f.projectID, channel, enabledLine)
}

func (f fakeNotifications) preferencesConfig(ids ...string) string {
	quoted := ""
	for i, id := range ids {
		if i > 0 {
			quoted += ", "
		}
		quoted += fmt.Sprintf("%q", id)
	}
	return f.providerConfig() + fmt.Sprintf(`
resource "circleci_project_notification_preferences" "builds" {
  organization_id        = %q
  project_id             = %q
  enabled_preference_ids = [%s]
}
`, f.orgID, f.projectID, quoted)
}

// projectPreferences reads the project's preferences straight from the fake.
func (f fakeNotifications) projectPreferences(t *testing.T) map[string]bool {
	t.Helper()
	prefs, err := f.service().ListProjectPreferences(context.TODO(), f.orgID, f.projectID)
	assert.NilError(t, err)
	out := map[string]bool{}
	for _, p := range prefs {
		out[p.ID] = p.Enabled
	}
	return out
}

func checkWelcomeMessages(f fakeNotifications, want int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if got := f.fake.SlackWelcomeMessages(); got != want {
			return fmt.Errorf("CircleCI posted %d welcome messages, want %d", got, want)
		}
		return nil
	}
}

func TestAccProjectSlackNotification(t *testing.T) {
	f := newFakeNotifications(t)
	f.fake.AddSlackChannel("C0B8D01J6RF", "builds")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			configs, err := f.service().ListProjectChannelConfigs(context.TODO(), f.orgID, f.projectID)
			if err != nil {
				return err
			}
			if len(configs) != 0 {
				return fmt.Errorf("%d channel configs left after destroy", len(configs))
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: f.slackConfig("C0B8D01J6RF", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("enabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("channel"), knownvalue.StringExact("C0B8D01J6RF")),
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("channel_name"), knownvalue.StringExact("builds")),
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("disable_reason"), knownvalue.Null()),
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("id"), knownvalue.NotNull()),
				},
				Check: checkWelcomeMessages(f, 1),
			},
			{
				// Re-applying an unchanged configuration must not send an
				// update, which would post the welcome message again.
				Config: f.slackConfig("C0B8D01J6RF", "true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkWelcomeMessages(f, 1),
			},
			{
				Config: f.slackConfig("C0NEW000001", "false"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(slackAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("enabled"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("channel"), knownvalue.StringExact("C0NEW000001")),
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("channel_name"), knownvalue.Null()),
				},
				Check: checkWelcomeMessages(f, 1),
			},
			{
				Config: f.slackConfig("C0NEW000001", "true"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(slackAddress, tfjsonpath.New("enabled"), knownvalue.Bool(true)),
				},
				Check: checkWelcomeMessages(f, 2),
			},
			{
				ResourceName:      slackAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(slackAddress, "organization_id", "project_id"),
			},
		},
	})
}

func TestAccProjectSlackNotificationDeletedOutOfBand(t *testing.T) {
	f := newFakeNotifications(t)
	config := f.slackConfig("C0B8D01J6RF", "")

	var id string
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					id = s.RootModule().Resources[slackAddress].Primary.ID
					return nil
				},
			},
			{
				PreConfig: func() {
					err := f.service().DeleteChannelConfig(context.TODO(), id)
					assert.NilError(t, err)
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(slackAddress, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccProjectSlackNotificationImportMissing(t *testing.T) {
	f := newFakeNotifications(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        f.slackConfig("C0B8D01J6RF", ""),
				ResourceName:  slackAddress,
				ImportState:   true,
				ImportStateId: f.orgID + "/" + f.projectID,
				ExpectError:   regexp.MustCompile(`no Slack notification`),
			},
		},
	})
}

func TestAccProjectSlackNotificationChannelName(t *testing.T) {
	f := newFakeNotifications(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      f.slackConfig("#builds", ""),
				ExpectError: regexp.MustCompile(`Slack channel ID`),
			},
		},
	})
}

func TestAccProjectNotificationPreferences(t *testing.T) {
	f := newFakeNotifications(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			got := f.projectPreferences(t)
			want := map[string]bool{
				fakecircle.NonConfigurablePreferenceID: true,
				prefDefaultBranch:                      false,
				prefApproval:                           false,
				prefSucceeded:                          false,
				prefFailed:                             false,
				prefInconclusive:                       false,
			}
			assert.Check(t, cmp.DeepEqual(got, want), "destroy turns every configurable preference off")
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: f.preferencesConfig(prefApproval, prefFailed),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(preferencesAddress, tfjsonpath.New("enabled_preference_ids"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact(prefApproval),
							knownvalue.StringExact(prefFailed),
						})),
				},
				Check: func(_ *terraform.State) error {
					got := f.projectPreferences(t)
					assert.Check(t, cmp.DeepEqual(got, map[string]bool{
						fakecircle.NonConfigurablePreferenceID: true,
						prefDefaultBranch:                      false,
						prefApproval:                           true,
						prefSucceeded:                          false,
						prefFailed:                             true,
						prefInconclusive:                       false,
					}), "preferences left out of the set are turned off")
					return nil
				},
			},
			{
				Config: f.preferencesConfig(prefDefaultBranch),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(preferencesAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(preferencesAddress, tfjsonpath.New("enabled_preference_ids"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(prefDefaultBranch)})),
				},
			},
			{
				ResourceName:      preferencesAddress,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importID(preferencesAddress, "organization_id", "project_id"),
			},
			{
				// A preference turned on outside Terraform is drift, and
				// applying turns it back off.
				PreConfig: func() {
					err := f.service().UpdateProjectPreferences(context.TODO(), f.orgID, f.projectID,
						map[string]bool{prefInconclusive: true})
					assert.NilError(t, err)
				},
				Config: f.preferencesConfig(prefDefaultBranch),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(preferencesAddress, plancheck.ResourceActionUpdate),
					},
				},
				Check: func(_ *terraform.State) error {
					got := f.projectPreferences(t)
					assert.Check(t, !got[prefInconclusive], "the out-of-band preference should be off again")
					return nil
				},
			},
		},
	})
}

func TestAccProjectNotificationPreferencesInvalid(t *testing.T) {
	f := newFakeNotifications(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      f.preferencesConfig("00000000-0000-0000-0000-000000000000"),
				ExpectError: regexp.MustCompile(`not a notification\s+preference\s+of\s+project`),
			},
			{
				Config:      f.preferencesConfig(fakecircle.NonConfigurablePreferenceID),
				ExpectError: regexp.MustCompile(`cannot\s+be\s+configured`),
			},
		},
	})
}

func TestAccNotificationPreferencesDataSource(t *testing.T) {
	f := newFakeNotifications(t)
	config := f.providerConfig() + fmt.Sprintf(`
data "circleci_notification_preferences" "builds" {
  organization_id = %q
  project_id      = %q
}
`, f.orgID, f.projectID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(preferencesDataSource, tfjsonpath.New("preferences"), knownvalue.ListSizeExact(6)),
					statecheck.ExpectKnownValue(preferencesDataSource, tfjsonpath.New("preferences").AtSliceIndex(1),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"id":                   knownvalue.StringExact(prefDefaultBranch),
							"name":                 knownvalue.StringExact("On default branch (failed, error, unauthorized)"),
							"channel":              knownvalue.StringExact("slack"),
							"group_name":           knownvalue.StringExact("Workflow Completed"),
							"section_name":         knownvalue.StringExact("Build"),
							"is_user_configurable": knownvalue.Bool(true),
							"is_enabled":           knownvalue.Bool(true),
						})),
					statecheck.ExpectKnownValue(preferencesDataSource,
						tfjsonpath.New("preferences").AtSliceIndex(0).AtMapKey("is_user_configurable"), knownvalue.Bool(false)),
				},
			},
		},
	})
}
