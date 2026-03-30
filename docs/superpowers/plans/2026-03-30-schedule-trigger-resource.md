# Schedule Trigger Support in `circleci_trigger` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing `circleci_trigger` Terraform resource to support `event_source_provider = "schedule"`, enabling cron-based pipeline triggers.

**Architecture:** Two-repo change. The `circleci-sdk-go` SDK gains a `common.Schedule` type and a `Trigger.Parameters` field. Then `trigger_resource.go` in the provider gets three new optional attributes (`cron_expression`, `attribution_actor`, `parameters`), a new `"schedule"` branch in Create validation, and schedule-aware Read/Update logic. No new resource type is introduced — schedule is a fourth value of the existing `event_source_provider` field.

**Tech Stack:** Go 1.25, Terraform Plugin Framework v1.19, `circleci-sdk-go` (local during dev), `gotest.tools/v3` (SDK tests), `terraform-plugin-testing` (provider acceptance tests)

---

## Context

The `circleci_trigger` resource already handles `github_app` and `webhook` providers with per-provider conditional fields in a single resource. Adding `schedule` follows the same pattern.

**SDK gaps to fill:**
- `common.EventSource` has no `Schedule` field
- `trigger.Trigger` has no `Parameters` field

**Provider changes needed (all in `trigger_resource.go`):**
- Three new schema attributes: `cron_expression`, `attribution_actor`, `parameters`
- Three new fields on `triggerResourceModel`
- New `"schedule"` case in Create validation
- Schedule fields wired into Create / Read / Update

**Known limitation:** The CircleCI read API does not return `attribution_actor` inside `event_source.schedule` — only `cron_expression` is present. Changes to `attribution_actor` made outside Terraform will not be detected on refresh.

---

## File Structure

### `circleci-sdk-go` (`/Users/benedetta/Code/CIRCLECI/circleci-sdk-go`)

| File | Change |
|------|--------|
| `common/models.go` | Add `Schedule` struct; add `Schedule Schedule` field to `EventSource` |
| `trigger/trigger.go` | Add `Parameters map[string]any` to `Trigger` |
| `trigger/trigger_test.go` | Add `TestFullScheduleTrigger` integration test |

### `terraform-provider-circleci` (`/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`)

| File | Change |
|------|--------|
| `go.mod` | Add `replace` directive pointing to local SDK (removed before final merge) |
| `internal/provider/trigger_resource.go` | Add fields to model + schema; extend Create, Read, Update |
| `internal/provider/trigger_resource_test.go` | Add `TestAccTriggerResourceSchedule` |
| `examples/resources/circleci_trigger/` | Add schedule example (new file alongside existing) |

---

## Task 1: Extend the SDK — add Schedule types

**Working directory:** `/Users/benedetta/Code/CIRCLECI/circleci-sdk-go`

**Files:**
- Modify: `common/models.go`
- Modify: `trigger/trigger.go`
- Modify: `trigger/trigger_test.go`

- [ ] **Step 1: Add `Schedule` struct and extend `EventSource` in `common/models.go`**

The complete updated file (add `Schedule` struct and `Schedule` field to `EventSource`; all other types unchanged):

```go
// nolint:revive // introduced before linter
package common

type Repo struct {
	FullName string `json:"full_name,omitempty"`
	// nolint:revive // introduced before linter
	ExternalId string `json:"external_id,omitempty"`
}

type Webhook struct {
	// nolint:revive // introduced before linter
	Url    string `json:"url,omitempty"`
	Sender string `json:"sender,omitempty"`
}

type Schedule struct {
	CronExpression   string `json:"cron_expression,omitempty"`
	AttributionActor string `json:"attribution_actor,omitempty"`
}

type ConfigSource struct {
	Provider string `json:"provider,omitempty"`
	Repo     Repo   `json:"repo,omitzero"`
	FilePath string `json:"file_path,omitempty"`
}

type CheckoutSource struct {
	Provider string `json:"provider,omitempty"`
	Repo     Repo   `json:"repo,omitzero"`
}

type EventSource struct {
	Provider string   `json:"provider,omitempty"`
	Repo     Repo     `json:"repo,omitzero"`
	Webhook  Webhook  `json:"webhook,omitzero"`
	Schedule Schedule `json:"schedule,omitzero"`
}

type PaginatedResponse[T any] struct {
	NextPageToken string `json:"next_page_token"`
	Items         []T    `json:"items"`
}

type VcsInfo struct {
	// nolint:revive // introduced before linter
	VcsUrl        string `json:"vcs_url"`
	Provider      string `json:"provider"`
	DefaultBranch string `json:"default_branch"`
}

type User struct {
	Login string `json:"login"`
}

type Scope struct {
	// nolint:revive // introduced before linter
	Id   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}
```

- [ ] **Step 2: Add `Parameters` field to `Trigger` in `trigger/trigger.go`**

Replace only the `Trigger` struct (one new line; all other code unchanged):

```go
type Trigger struct {
	ID          string             `json:"id,omitempty"`
	CreatedAt   string             `json:"created_at,omitempty"`
	CheckoutRef string             `json:"checkout_ref,omitempty"`
	ConfigRef   string             `json:"config_ref,omitempty"`
	EventSource common.EventSource `json:"event_source,omitzero"`
	EventName   string             `json:"event_name,omitempty"`
	EventPreset string             `json:"event_preset,omitempty"`
	Disabled    *bool              `json:"disabled,omitempty"`
	Parameters  map[string]any     `json:"parameters,omitempty"`
}
```

- [ ] **Step 3: Add `TestFullScheduleTrigger` to `trigger/trigger_test.go`**

Add a new constant and test at the end of the file. **`knownSchedulePipelineID` must be a pipeline definition that supports schedule triggers — fill in the correct UUID before running.**

```go
const knownSchedulePipelineID = "FILL_IN_PIPELINE_DEFINITION_ID_THAT_SUPPORTS_SCHEDULE_TRIGGERS"

func TestFullScheduleTrigger(t *testing.T) {
	ctx := context.TODO()
	c := integrationtest.Client(t)
	triggerService := NewTriggerService(c)

	newTrigger := Trigger{
		EventName:   "Test schedule trigger",
		CheckoutRef: "main",
		ConfigRef:   "main",
		Disabled:    common.Bool(false),
		EventSource: common.EventSource{
			Provider: "schedule",
			Schedule: common.Schedule{
				CronExpression:   "0 1 * * *",
				AttributionActor: "current",
			},
		},
		Parameters: map[string]any{"env": "staging"},
	}

	created, err := triggerService.Create(ctx, newTrigger, knownProjectID, knownSchedulePipelineID)
	assert.Assert(t, err)
	assert.Check(t, created.ID != "")
	assert.Check(t, cmp.Equal(created.EventSource.Provider, "schedule"))

	fetched, err := triggerService.Get(ctx, knownProjectID, created.ID)
	assert.Assert(t, err)
	assert.Check(t, cmp.Equal(fetched.EventSource.Schedule.CronExpression, "0 1 * * *"))

	_, err = triggerService.Update(ctx, Trigger{
		EventName: "Updated schedule trigger",
		EventSource: common.EventSource{
			Schedule: common.Schedule{CronExpression: "0 2 * * *"},
		},
	}, knownProjectID, created.ID)
	assert.Assert(t, err)

	err = triggerService.Delete(ctx, knownProjectID, created.ID)
	assert.Assert(t, err)

	deleted, err := triggerService.Get(ctx, knownProjectID, created.ID)
	assert.Assert(t, err != nil)
	assert.Check(t, cmp.Nil(deleted))
}
```

- [ ] **Step 4: Verify existing SDK tests still pass (no regressions)**

```bash
cd /Users/benedetta/Code/CIRCLECI/circleci-sdk-go
CIRCLE_TOKEN=<your-token> go test ./trigger/... -run "TestFullTrigger$|TestFullTriggerNew" -v
```

Expected: both tests PASS.

- [ ] **Step 5: Verify the SDK builds**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit SDK changes**

```bash
git add common/models.go trigger/trigger.go trigger/trigger_test.go
git commit -m "feat: add Schedule type and Parameters field for schedule trigger support"
```

---

## Task 2: Point the Terraform provider at the local SDK

**Working directory:** `/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add a `replace` directive to `go.mod`**

Add at the end of `go.mod` (after the last closing parenthesis):

```
replace github.com/CircleCI-Public/circleci-sdk-go => ../circleci-sdk-go
```

- [ ] **Step 2: Sync dependencies**

```bash
go mod tidy
```

Expected: exits 0. `go.sum` is updated.

- [ ] **Step 3: Verify the provider still builds**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: point SDK at local clone for schedule trigger development"
```

---

## Task 3: Write the failing acceptance test (TDD)

**Working directory:** `/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`

**Files:**
- Modify: `internal/provider/trigger_resource_test.go`

Write the test before implementing, so we can watch it fail first.

**`testSchedulePipelineID` must be a pipeline definition UUID that supports schedule triggers.**

- [ ] **Step 1: Add the schedule test and config helper to `trigger_resource_test.go`**

Add after the last function in the file:

```go
func TestAccTriggerResourceSchedule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccTriggerResourceScheduleConfig(
					"61169e84-93ee-415d-8d65-ddf6dc0d2939",
					"FILL_IN_PIPELINE_DEFINITION_ID_THAT_SUPPORTS_SCHEDULE_TRIGGERS",
					"Nightly build",
					"0 1 * * *",
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_schedule",
						tfjsonpath.New("event_source_provider"),
						knownvalue.StringExact("schedule"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_schedule",
						tfjsonpath.New("event_name"),
						knownvalue.StringExact("Nightly build"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_schedule",
						tfjsonpath.New("cron_expression"),
						knownvalue.StringExact("0 1 * * *"),
					),
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_schedule",
						tfjsonpath.New("disabled"),
						knownvalue.Bool(false),
					),
				},
			},
			// Update — change cron expression
			{
				Config: testAccTriggerResourceScheduleConfig(
					"61169e84-93ee-415d-8d65-ddf6dc0d2939",
					"FILL_IN_PIPELINE_DEFINITION_ID_THAT_SUPPORTS_SCHEDULE_TRIGGERS",
					"Nightly build",
					"0 2 * * *",
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"circleci_trigger.test_trigger_schedule",
						tfjsonpath.New("cron_expression"),
						knownvalue.StringExact("0 2 * * *"),
					),
				},
			},
			// ImportState
			{
				ResourceName:            "circleci_trigger.test_trigger_schedule",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"pipeline_id", "attribution_actor"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					triggerID, found := s.RootModule().Resources["circleci_trigger.test_trigger_schedule"].Primary.Attributes["id"]
					if !found {
						return "", errors.New("attribute id not found")
					}
					projectID, found := s.RootModule().Resources["circleci_trigger.test_trigger_schedule"].Primary.Attributes["project_id"]
					if !found {
						return "", errors.New("attribute project_id not found")
					}
					return fmt.Sprintf("%s/%s", projectID, triggerID), nil
				},
			},
		},
	})
}

func testAccTriggerResourceScheduleConfig(projectID, pipelineID, eventName, cron string) string {
	return fmt.Sprintf(`
resource "circleci_trigger" "test_trigger_schedule" {
  project_id           = %[1]q
  pipeline_id          = %[2]q
  event_source_provider = "schedule"
  event_name           = %[3]q
  cron_expression      = %[4]q
  checkout_ref         = "main"
  config_ref           = "main"
}
`, projectID, pipelineID, eventName, cron)
}
```

- [ ] **Step 2: Verify the file compiles (it will fail to compile because `cron_expression` doesn't exist in the schema yet)**

```bash
go build ./...
```

Expected: compile error mentioning `cron_expression` unknown attribute. This confirms TDD is working — the test drives the implementation.

---

## Task 4: Extend `trigger_resource.go` with schedule support

**Working directory:** `/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`

**Files:**
- Modify: `internal/provider/trigger_resource.go`

- [ ] **Step 1: Add three fields to `triggerResourceModel`**

In `trigger_resource.go`, find the `triggerResourceModel` struct and add three fields at the end:

```go
type triggerResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	ProjectId                 types.String `tfsdk:"project_id"`
	PipelineId                types.String `tfsdk:"pipeline_id"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	CheckoutRef               types.String `tfsdk:"checkout_ref"`
	ConfigRef                 types.String `tfsdk:"config_ref"`
	EventSourceProvider       types.String `tfsdk:"event_source_provider"`
	EventSourceRepoFullName   types.String `tfsdk:"event_source_repo_full_name"`
	EventSourceRepoExternalId types.String `tfsdk:"event_source_repo_external_id"`
	EventSourceWebHookUrl     types.String `tfsdk:"event_source_web_hook_url"`
	EventSourceWebHookSender  types.String `tfsdk:"event_source_web_hook_sender"`
	EventPreset               types.String `tfsdk:"event_preset"`
	EventName                 types.String `tfsdk:"event_name"`
	Disabled                  types.Bool   `tfsdk:"disabled"`
	CronExpression            types.String `tfsdk:"cron_expression"`
	AttributionActor          types.String `tfsdk:"attribution_actor"`
	Parameters                types.Map    `tfsdk:"parameters"`
}
```

- [ ] **Step 2: Add three attributes to the Schema**

In the `Schema` method, add to the `Attributes` map after the `"disabled"` entry:

```go
			"cron_expression": schema.StringAttribute{
				MarkdownDescription: "A cron expression defining when the trigger fires (e.g. `*/5 * * * *` for every 5 minutes). Required when `event_source_provider` is `schedule`.",
				Optional:            true,
			},
			"attribution_actor": schema.StringAttribute{
				MarkdownDescription: "The actor to attribute pipeline runs to. One of `current` or `system`. Only applicable when `event_source_provider` is `schedule`. If omitted the API defaults to `current`. Note: not returned by the read API — external changes will not be detected.",
				Optional:            true,
			},
			"parameters": schema.MapAttribute{
				MarkdownDescription: "Pipeline parameters to pass to triggered pipeline runs. Values must be strings. Only applicable when `event_source_provider` is `schedule`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
```

Also update the import block — add `"github.com/hashicorp/terraform-plugin-framework/attr"` (needed for building `types.Map` values).

- [ ] **Step 3: Add `"schedule"` case to the `Create` validation switch**

In `Create`, find the `switch circleCiTerrformTriggerResource.EventSourceProvider.ValueString()` block. Add a `"schedule"` case before `default` and update the default error message:

```go
	case "schedule":
		if circleCiTerrformTriggerResource.CronExpression.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires a cron_expression",
			)
			return
		}
		if circleCiTerrformTriggerResource.EventName.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires an event_name",
			)
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Error creating CircleCI trigger",
			"CircleCI trigger has an unexpected event source provider: expected one of github_app, webhook, or schedule",
		)
		return
```

- [ ] **Step 4: Wire `Schedule` and `Parameters` into the Create request**

In `Create`, after the existing `newEventSource` construction, add the schedule block:

```go
	// New EventSource
	newEventSource := common.EventSource{
		Provider: circleCiTerrformTriggerResource.EventSourceProvider.ValueString(),
		Repo:     newRepo,
		Webhook:  newWebHook,
	}
	if circleCiTerrformTriggerResource.EventSourceProvider.ValueString() == "schedule" {
		newEventSource.Schedule = common.Schedule{
			CronExpression:   circleCiTerrformTriggerResource.CronExpression.ValueString(),
			AttributionActor: circleCiTerrformTriggerResource.AttributionActor.ValueString(),
		}
	}
```

And after the `newTrigger` construction, add parameters:

```go
	if !circleCiTerrformTriggerResource.Parameters.IsNull() && !circleCiTerrformTriggerResource.Parameters.IsUnknown() {
		params := make(map[string]string)
		resp.Diagnostics.Append(circleCiTerrformTriggerResource.Parameters.ElementsAs(ctx, &params, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		paramsAny := make(map[string]any, len(params))
		for k, v := range params {
			paramsAny[k] = v
		}
		newTrigger.Parameters = paramsAny
	}
```

After the existing post-create state mapping (after `r.client.Get` call), add:

```go
	// Schedule-specific fields
	if readTrigger.EventSource.Schedule.CronExpression != "" {
		circleCiTerrformTriggerResource.CronExpression = types.StringValue(readTrigger.EventSource.Schedule.CronExpression)
	}
	// attribution_actor is not in the API response — the plan value is already in circleCiTerrformTriggerResource.AttributionActor
	circleCiTerrformTriggerResource.Parameters = triggerParametersFromAny(readTrigger.Parameters)
```

- [ ] **Step 5: Wire schedule fields into the `Read` function**

In `Read`, find the `switch triggerState.EventSourceProvider.ValueString()` block and add a `"schedule"` case:

```go
	switch triggerState.EventSourceProvider.ValueString() {
	case "webhook":
		triggerState.EventSourceWebHookSender = types.StringValue(readTrigger.EventSource.Webhook.Sender)
	case "github_app":
	case "schedule":
		if readTrigger.EventSource.Schedule.CronExpression != "" {
			triggerState.CronExpression = types.StringValue(readTrigger.EventSource.Schedule.CronExpression)
		} else {
			triggerState.CronExpression = types.StringNull()
		}
		// attribution_actor is not in the API read response — preserve existing state value unchanged
		triggerState.Parameters = triggerParametersFromAny(readTrigger.Parameters)
	}
```

- [ ] **Step 6: Wire schedule fields into the `Update` function**

In `Update`, after the existing `newEventSource` construction, add:

```go
	if state.EventSourceProvider.ValueString() == "schedule" {
		newEventSource.Schedule = common.Schedule{
			CronExpression:   state.CronExpression.ValueString(),
			AttributionActor: state.AttributionActor.ValueString(),
		}
	}
```

And after the `updates` trigger construction, add parameters:

```go
	if !state.Parameters.IsNull() && !state.Parameters.IsUnknown() {
		params := make(map[string]string)
		resp.Diagnostics.Append(state.Parameters.ElementsAs(ctx, &params, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		paramsAny := make(map[string]any, len(params))
		for k, v := range params {
			paramsAny[k] = v
		}
		updates.Parameters = paramsAny
	}
```

After the existing post-update state mapping, add:

```go
	// Schedule-specific fields
	if updatedTrigger.EventSource.Schedule.CronExpression != "" {
		state.CronExpression = types.StringValue(updatedTrigger.EventSource.Schedule.CronExpression)
	}
	// attribution_actor not in response — plan value already in state.AttributionActor
	state.Parameters = triggerParametersFromAny(updatedTrigger.Parameters)
```

- [ ] **Step 7: Add the `triggerParametersFromAny` helper at the bottom of the file**

Add after the existing `isApiNotFoundError` function:

```go
// triggerParametersFromAny converts an API map[string]any to a Terraform types.Map.
// Returns a null map when params is empty, so that unset parameters in config
// don't drift against an empty API response.
func triggerParametersFromAny(params map[string]any) types.Map {
	if len(params) == 0 {
		return types.MapNull(types.StringType)
	}
	elements := make(map[string]attr.Value, len(params))
	for k, v := range params {
		elements[k] = types.StringValue(fmt.Sprintf("%v", v))
	}
	result, _ := types.MapValue(types.StringType, elements)
	return result
}
```

- [ ] **Step 8: Verify the provider builds**

```bash
go build ./...
```

Expected: no errors. The test file from Task 3 should now also compile.

- [ ] **Step 9: Commit**

```bash
git add internal/provider/trigger_resource.go
git commit -m "feat: add schedule trigger support to circleci_trigger resource"
```

---

## Task 5: Run the acceptance tests

**Working directory:** `/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`

- [ ] **Step 1: Fill in the pipeline definition ID in the test file**

In `trigger_resource_test.go`, replace both occurrences of:
```
"FILL_IN_PIPELINE_DEFINITION_ID_THAT_SUPPORTS_SCHEDULE_TRIGGERS"
```
with the real UUID of a pipeline definition that supports schedule triggers.

- [ ] **Step 2: Run the schedule trigger test**

```bash
TF_ACC=1 CIRCLE_TOKEN=<your-token> go test ./internal/provider/... -run TestAccTriggerResourceSchedule -v -timeout 120s
```

Expected: PASS across all three steps (create, update, import).

- [ ] **Step 3: Run the full existing test suite to check for regressions**

```bash
TF_ACC=1 CIRCLE_TOKEN=<your-token> go test ./internal/provider/... -run "TestAccTriggerResource" -v -timeout 300s
```

Expected: `TestAccTriggerResourceGithub`, `TestAccTriggerResourceWebhook`, and `TestAccTriggerResourceSchedule` all PASS.

- [ ] **Step 4: Commit the test file with the real pipeline ID**

```bash
git add internal/provider/trigger_resource_test.go
git commit -m "test: add schedule trigger acceptance test"
```

---

## Task 6: Add an example and finalize

**Working directory:** `/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`

- [ ] **Step 1: Add a schedule trigger example**

Create `examples/resources/circleci_trigger/schedule.tf`:

```hcl
# Schedule trigger — runs the pipeline every day at 01:00 UTC
resource "circleci_trigger" "nightly" {
  project_id            = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
  pipeline_id           = "fefb451c-9966-4b75-b555-d4d94d7116ef"
  event_source_provider = "schedule"
  event_name            = "Nightly build"
  cron_expression       = "0 1 * * *"
  checkout_ref          = "main"
  config_ref            = "main"
  attribution_actor     = "current"

  parameters = {
    deploy_env = "staging"
  }
}
```

- [ ] **Step 2: Format the example**

```bash
terraform fmt examples/resources/circleci_trigger/schedule.tf
```

- [ ] **Step 3: Commit**

```bash
git add examples/resources/circleci_trigger/schedule.tf
git commit -m "docs: add schedule trigger example for circleci_trigger"
```

---

## Task 7: Finalize — tag SDK, remove replace directive

**SDK working directory:** `/Users/benedetta/Code/CIRCLECI/circleci-sdk-go`
**Provider working directory:** `/Users/benedetta/Code/CIRCLECI/terraform-provider-circleci`

- [ ] **Step 1: Push and tag the SDK**

```bash
cd /Users/benedetta/Code/CIRCLECI/circleci-sdk-go
git push origin main
git tag v0.1.0   # use the next appropriate semver for this repo
git push origin v0.1.0
```

- [ ] **Step 2: Update the provider to use the tagged SDK**

```bash
cd /Users/benedetta/Code/CIRCLECI/terraform-provider-circleci
go get github.com/CircleCI-Public/circleci-sdk-go@v0.1.0
```

- [ ] **Step 3: Remove the `replace` directive from `go.mod`**

Delete the line added in Task 2:
```
replace github.com/CircleCI-Public/circleci-sdk-go => ../circleci-sdk-go
```

- [ ] **Step 4: Tidy and build**

```bash
go mod tidy
go build ./...
```

Expected: both succeed.

- [ ] **Step 5: Final test run**

```bash
TF_ACC=1 CIRCLE_TOKEN=<your-token> go test ./internal/provider/... -run "TestAccTriggerResource" -v -timeout 300s
```

Expected: all three trigger tests PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: update SDK to tagged version with schedule trigger support"
```

---

## Verification Checklist

- [ ] `go build ./...` passes in both repos
- [ ] `TestAccTriggerResourceGithub` still passes (no regression)
- [ ] `TestAccTriggerResourceWebhook` still passes (no regression)
- [ ] `TestAccTriggerResourceSchedule` passes (create, update, import)
- [ ] `go.mod` has no `replace` directive before opening PRs
- [ ] `terraform fmt` produces no changes in `examples/`
