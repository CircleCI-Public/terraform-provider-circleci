// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/common"
	"github.com/CircleCI-Public/circleci-sdk-go/trigger"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &triggerResource{}
	_ resource.ResourceWithConfigure   = &triggerResource{}
	_ resource.ResourceWithImportState = &triggerResource{}
)

// triggerResourceModel maps the output schema.
type triggerResourceModel struct {
	Id                                  types.String `tfsdk:"id"`
	ProjectId                           types.String `tfsdk:"project_id"`
	PipelineId                          types.String `tfsdk:"pipeline_id"`
	CreatedAt                           types.String `tfsdk:"created_at"`
	CheckoutRef                         types.String `tfsdk:"checkout_ref"`
	ConfigRef                           types.String `tfsdk:"config_ref"`
	EventSourceProvider                 types.String `tfsdk:"event_source_provider"`
	EventSourceRepoFullName             types.String `tfsdk:"event_source_repo_full_name"`
	EventSourceRepoExternalId           types.String `tfsdk:"event_source_repo_external_id"`
	EventSourceWebHookUrl               types.String `tfsdk:"event_source_web_hook_url"`
	EventSourceWebHookSender            types.String `tfsdk:"event_source_web_hook_sender"`
	EventSourceScheduleCronExpression   types.String `tfsdk:"event_source_schedule_cron_expression"`
	EventSourceScheduleAttributionActor types.String `tfsdk:"event_source_schedule_attribution_actor"`
	EventPreset                         types.String `tfsdk:"event_preset"`
	EventName                           types.String `tfsdk:"event_name"`
	Disabled                            types.Bool   `tfsdk:"disabled"`
}

// NewTriggerResource is a helper function to simplify the provider implementation.
func NewTriggerResource() resource.Resource {
	return &triggerResource{}
}

// triggerResource is the resource implementation.
type triggerResource struct {
	client *trigger.TriggerService
}

// Metadata returns the resource type name.
func (r *triggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

// Schema defines the schema for the resource.
func (r *triggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci trigger",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					// This is the CRITICAL line. It suppresses the drift by telling TF
					// to ignore the 'unknown' value coming from the Read and use the prior state.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "project_id of the circleci trigger",
				Required:            true,
			},
			"pipeline_id": schema.StringAttribute{
				MarkdownDescription: "pipeline_id of the circleci trigger",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci trigger",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checkout_ref": schema.StringAttribute{
				MarkdownDescription: "The ref to use when checking out code for pipeline runs created from this trigger. Always required when `event_source_provider` is `webhook` or `schedule`. When `event_source_provider` is `github_app` or `github_server`, only expected if the event source repository differs from the checkout source repository of the associated pipeline definition. Otherwise, must be omitted.",
				Optional:            true,
			},
			"config_ref": schema.StringAttribute{
				MarkdownDescription: "The ref to use when fetching configuration for pipeline runs created from this trigger. Always required when `event_source_provider` is `webhook` or `schedule`. When `event_source_provider` is `github_app` or `github_server`, only expected if the event source repository differs from the config source repository of the associated pipeline definition. Otherwise, must be omitted.",
				Optional:            true,
			},
			"event_source_provider": schema.StringAttribute{
				MarkdownDescription: "event_source_provider of the circleci trigger",
				Required:            true,
			},
			"event_source_repo_full_name": schema.StringAttribute{
				MarkdownDescription: "event_source_repo_full_name of the circleci trigger",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"event_source_repo_external_id": schema.StringAttribute{
				MarkdownDescription: "event_source_repo_external_id of the circleci trigger",
				Optional:            true,
			},
			"event_source_web_hook_url": schema.StringAttribute{
				MarkdownDescription: "event_source_web_hook_url of the circleci trigger",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"event_source_web_hook_sender": schema.StringAttribute{
				MarkdownDescription: "event_source_web_hook_sender of the circleci trigger",
				Optional:            true,
			},
			"event_source_schedule_cron_expression": schema.StringAttribute{
				MarkdownDescription: "Cron expression for the schedule event source. Required when event_source_provider is schedule.",
				Optional:            true,
				Validators:          []validator.String{CronExpressionValidator()},
			},
			"event_source_schedule_attribution_actor": schema.StringAttribute{
				MarkdownDescription: "Attribution actor for the schedule event source. Required when event_source_provider is schedule. Must be \"system\" or \"current\".",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{stringvalidator.OneOf("system", "current")},
			},
			"event_preset": schema.StringAttribute{
				MarkdownDescription: "event_preset of the circleci trigger",
				Optional:            true,
			},
			"event_name": schema.StringAttribute{
				MarkdownDescription: "event_name of the circleci trigger",
				Optional:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "disabled of the circleci trigger",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *triggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var circleCiTerrformTriggerResource triggerResourceModel
	diags := req.Plan.Get(ctx, &circleCiTerrformTriggerResource)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch circleCiTerrformTriggerResource.EventSourceProvider.ValueString() {
	case "github_app", "github_server":
		if !isValidEventPreset(circleCiTerrformTriggerResource.EventPreset.ValueString()) {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with "+circleCiTerrformTriggerResource.EventSourceProvider.ValueString()+" provider has an unexpected event_preset",
			)
			return
		}
		if !circleCiTerrformTriggerResource.EventName.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with "+circleCiTerrformTriggerResource.EventSourceProvider.ValueString()+" provider does not support event_name",
			)
			return
		}
	case "webhook":
		if circleCiTerrformTriggerResource.EventSourceWebHookUrl.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with webhook provider has an unexpected event source web hook url",
			)
			return
		}
		if circleCiTerrformTriggerResource.EventName.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with webhook provider requires an event_name",
			)
			return
		}
		if circleCiTerrformTriggerResource.EventSourceWebHookSender.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with webhook provider requires a Webhook Sender",
			)
			return
		}
	case "schedule":
		if circleCiTerrformTriggerResource.EventName.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires an event_name",
			)
			return
		}
		if circleCiTerrformTriggerResource.CheckoutRef.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires checkout_ref",
			)
			return
		}
		if circleCiTerrformTriggerResource.ConfigRef.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires config_ref",
			)
			return
		}
		if circleCiTerrformTriggerResource.EventSourceScheduleCronExpression.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires event_source_schedule_cron_expression",
			)
			return
		}
		if circleCiTerrformTriggerResource.EventSourceScheduleAttributionActor.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with schedule provider requires event_source_schedule_attribution_actor",
			)
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Error creating CircleCI trigger",
			"CircleCI trigger has an unexpected event source provider: should be either github_app, github_server, webhook, or schedule",
		)
		return
	}

	// New Webhook
	newWebHook := common.Webhook{
		Url:    circleCiTerrformTriggerResource.EventSourceWebHookUrl.ValueString(),
		Sender: circleCiTerrformTriggerResource.EventSourceWebHookSender.ValueString(),
	}

	// New Repo
	newRepo := common.Repo{
		FullName:   "",
		ExternalId: circleCiTerrformTriggerResource.EventSourceRepoExternalId.ValueString(),
	}

	// New Schedule
	newSchedule := common.Schedule{
		CronExpression:   circleCiTerrformTriggerResource.EventSourceScheduleCronExpression.ValueString(),
		AttributionActor: circleCiTerrformTriggerResource.EventSourceScheduleAttributionActor.ValueString(),
	}

	// New EventSource
	newEventSource := common.EventSource{
		Provider: circleCiTerrformTriggerResource.EventSourceProvider.ValueString(),
		Repo:     newRepo,
		Webhook:  newWebHook,
		Schedule: newSchedule,
	}

	// New Trigger
	disabled := circleCiTerrformTriggerResource.Disabled.ValueBool()
	newTrigger := trigger.Trigger{
		EventName:   circleCiTerrformTriggerResource.EventName.ValueString(),
		CheckoutRef: circleCiTerrformTriggerResource.CheckoutRef.ValueString(),
		ConfigRef:   circleCiTerrformTriggerResource.ConfigRef.ValueString(),
		EventSource: newEventSource,
		EventPreset: circleCiTerrformTriggerResource.EventPreset.ValueString(),
		Disabled:    &disabled,
	}

	// Create new Trigger
	newReturnedTrigger, err := r.client.Create(
		ctx,
		newTrigger,
		circleCiTerrformTriggerResource.ProjectId.ValueString(),
		circleCiTerrformTriggerResource.PipelineId.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI trigger",
			"Could not create CircleCI trigger, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	circleCiTerrformTriggerResource.Id = types.StringValue(newReturnedTrigger.ID)
	circleCiTerrformTriggerResource.PipelineId = types.StringValue(circleCiTerrformTriggerResource.PipelineId.ValueString())
	if newReturnedTrigger.CheckoutRef != "" {
		circleCiTerrformTriggerResource.CheckoutRef = types.StringValue(newReturnedTrigger.CheckoutRef)
	}
	if newReturnedTrigger.ConfigRef != "" {
		circleCiTerrformTriggerResource.ConfigRef = types.StringValue(newReturnedTrigger.ConfigRef)
	}
	circleCiTerrformTriggerResource.EventSourceProvider = types.StringValue(newReturnedTrigger.EventSource.Provider)
	if newReturnedTrigger.EventSource.Repo.FullName == "" {
		circleCiTerrformTriggerResource.EventSourceRepoFullName = types.StringNull()
	} else {
		circleCiTerrformTriggerResource.EventSourceRepoFullName = types.StringValue(newReturnedTrigger.EventSource.Repo.FullName)
	}

	if newReturnedTrigger.EventSource.Repo.ExternalId != "" {
		circleCiTerrformTriggerResource.EventSourceRepoExternalId = types.StringValue(newReturnedTrigger.EventSource.Repo.ExternalId)
	}
	circleCiTerrformTriggerResource.EventSourceWebHookUrl = types.StringValue(newReturnedTrigger.EventSource.Webhook.Url)
	if newReturnedTrigger.EventPreset != "" {
		circleCiTerrformTriggerResource.EventPreset = types.StringValue(newReturnedTrigger.EventPreset)
	}
	if circleCiTerrformTriggerResource.EventSourceProvider.ValueString() == "webhook" && circleCiTerrformTriggerResource.EventName.ValueString() != "" {
		circleCiTerrformTriggerResource.EventName = types.StringValue(newReturnedTrigger.EventName)
	}
	if newReturnedTrigger.EventSource.Schedule.CronExpression != "" {
		circleCiTerrformTriggerResource.EventSourceScheduleCronExpression = types.StringValue(newReturnedTrigger.EventSource.Schedule.CronExpression)
	} else {
		circleCiTerrformTriggerResource.EventSourceScheduleCronExpression = types.StringNull()
	}
	// For schedule triggers, preserve the user's input value. The API may transform aliases
	// like "system" to a UUID, which would cause perpetual drift if stored in state.
	if circleCiTerrformTriggerResource.EventSourceProvider.ValueString() != "schedule" {
		circleCiTerrformTriggerResource.EventSourceScheduleAttributionActor = types.StringNull()
	}
	circleCiTerrformTriggerResource.Disabled = types.BoolValue(*newReturnedTrigger.Disabled)

	readTrigger, err := r.client.Get(ctx, circleCiTerrformTriggerResource.ProjectId.ValueString(), newReturnedTrigger.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed retrieving", err.Error())
		// Cleanup may be required here (e.g., Delete the resource if it failed to settle)
		return
	}
	circleCiTerrformTriggerResource.CreatedAt = types.StringValue(readTrigger.CreatedAt)
	if readTrigger.EventSource.Repo.FullName == "" {
		circleCiTerrformTriggerResource.EventSourceRepoFullName = types.StringNull()
	} else {
		circleCiTerrformTriggerResource.EventSourceRepoFullName = types.StringValue(readTrigger.EventSource.Repo.FullName)
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, circleCiTerrformTriggerResource)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *triggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var triggerState triggerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &triggerState)...)
	if resp.Diagnostics.HasError() {
		return // Stop immediately if error occurred during state retrieval
	}

	if triggerState.Id.IsNull() || triggerState.Id.IsUnknown() {
		// ID is lost, meaning the resource is unmanaged or deleted.
		resp.State.RemoveResource(ctx)
		return
	}

	if triggerState.ProjectId.IsNull() || triggerState.ProjectId.IsUnknown() {
		resp.State.RemoveResource(ctx)
		return
	}

	readTrigger, err := r.client.Get(ctx, triggerState.ProjectId.ValueString(), triggerState.Id.ValueString())
	if err != nil {
		if isApiNotFoundError(err) {
			// This is the line that must be hit when the resource is gone.
			resp.State.RemoveResource(ctx)
			return // Successfully removed resource from state
		}

		// Standard error return path
		resp.Diagnostics.AddError("Error Reading Trigger", fmt.Sprintf("API error during read: %s", err.Error()))
		return
	}

	// Map response body to model
	triggerState.Id = types.StringValue(readTrigger.ID)
	triggerState.CreatedAt = types.StringValue(readTrigger.CreatedAt)

	if readTrigger.CheckoutRef == "" {
		triggerState.CheckoutRef = types.StringNull()
	} else {
		triggerState.CheckoutRef = types.StringValue(readTrigger.CheckoutRef)
	}

	if readTrigger.ConfigRef == "" {
		triggerState.ConfigRef = types.StringNull()
	} else {
		triggerState.ConfigRef = types.StringValue(readTrigger.ConfigRef)
	}

	if readTrigger.EventSource.Provider == "" {
		triggerState.EventSourceProvider = types.StringNull()
	} else {
		triggerState.EventSourceProvider = types.StringValue(readTrigger.EventSource.Provider)
	}

	if readTrigger.EventSource.Repo.FullName == "" {
		triggerState.EventSourceRepoFullName = types.StringNull()
	} else {
		triggerState.EventSourceRepoFullName = types.StringValue(readTrigger.EventSource.Repo.FullName)
	}
	triggerState.EventSourceWebHookUrl = types.StringValue(readTrigger.EventSource.Webhook.Url)
	switch triggerState.EventSourceProvider.ValueString() {
	case "webhook":
		triggerState.EventSourceWebHookSender = types.StringValue(readTrigger.EventSource.Webhook.Sender)
	case "github_app", "github_server", "schedule":
	}

	if readTrigger.EventName == "" {
		triggerState.EventName = types.StringNull()
	} else {
		triggerState.EventName = types.StringValue(readTrigger.EventName)
	}

	if readTrigger.EventPreset == "" {
		triggerState.EventPreset = types.StringNull()
	} else {
		triggerState.EventPreset = types.StringValue(readTrigger.EventPreset)
	}

	if readTrigger.EventSource.Repo.ExternalId == "" {
		triggerState.EventSourceRepoExternalId = types.StringNull()
	} else {
		triggerState.EventSourceRepoExternalId = types.StringValue(readTrigger.EventSource.Repo.ExternalId)
	}

	if readTrigger.EventSource.Schedule.CronExpression == "" {
		triggerState.EventSourceScheduleCronExpression = types.StringNull()
	} else {
		triggerState.EventSourceScheduleCronExpression = types.StringValue(readTrigger.EventSource.Schedule.CronExpression)
	}

	// Preserve the prior state value for attribution_actor so aliases like "system" don't drift
	// to their resolved UUID. Only set from the API when the state has no value (e.g. import).
	if triggerState.EventSourceScheduleAttributionActor.IsNull() || triggerState.EventSourceScheduleAttributionActor.IsUnknown() {
		if readTrigger.EventSource.Schedule.AttributionActor.Id == "" {
			triggerState.EventSourceScheduleAttributionActor = types.StringNull()
		} else {
			triggerState.EventSourceScheduleAttributionActor = types.StringValue(readTrigger.EventSource.Schedule.AttributionActor.Id)
		}
	}

	if readTrigger.Disabled == nil || !*readTrigger.Disabled {
		triggerState.Disabled = types.BoolValue(false)
	} else {
		triggerState.Disabled = types.BoolValue(true)
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &triggerState)...)
	// Always check for errors after the final Set
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *triggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state triggerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare the new trigger
	newWebHook := common.Webhook{
		Url:    state.EventSourceWebHookUrl.ValueString(),
		Sender: state.EventSourceWebHookSender.ValueString(),
	}
	// New Repo
	newRepo := common.Repo{
		FullName:   "",
		ExternalId: state.EventSourceRepoExternalId.ValueString(),
	}
	// New Schedule
	newSchedule := common.Schedule{
		CronExpression:   state.EventSourceScheduleCronExpression.ValueString(),
		AttributionActor: state.EventSourceScheduleAttributionActor.ValueString(),
	}

	// New EventSource
	newEventSource := common.EventSource{
		Provider: state.EventSourceProvider.ValueString(),
		Repo:     newRepo,
		Webhook:  newWebHook,
		Schedule: newSchedule,
	}

	// New Trigger
	disabled := state.Disabled.ValueBool()
	updates := trigger.Trigger{
		EventName:   state.EventName.ValueString(),
		CheckoutRef: state.CheckoutRef.ValueString(),
		ConfigRef:   state.ConfigRef.ValueString(),
		EventSource: newEventSource,
		EventPreset: state.EventPreset.ValueString(),
		Disabled:    &disabled,
	}

	// update the trigger
	updatedTrigger, err := r.client.Update(ctx, updates, state.ProjectId.ValueString(), state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update CircleCI trigger with id "+state.Id.ValueString()+" and project id "+state.ProjectId.ValueString(),
			err.Error(),
		)
		return
	}

	// update state
	state.Id = types.StringValue(updatedTrigger.ID)
	state.CheckoutRef = types.StringValue(updatedTrigger.CheckoutRef)
	state.ConfigRef = types.StringValue(updatedTrigger.ConfigRef)
	state.EventSourceProvider = types.StringValue(updatedTrigger.EventSource.Provider)
	if updatedTrigger.EventSource.Repo.FullName == "" {
		state.EventSourceRepoFullName = types.StringNull()
	} else {
		state.EventSourceRepoFullName = types.StringValue(updatedTrigger.EventSource.Repo.FullName)
	}
	if updatedTrigger.EventSource.Repo.ExternalId == "" {
		state.EventSourceRepoExternalId = types.StringNull()
	} else {
		state.EventSourceRepoExternalId = types.StringValue(updatedTrigger.EventSource.Repo.ExternalId)
	}
	state.EventSourceWebHookUrl = types.StringValue(updatedTrigger.EventSource.Webhook.Url)
	if updatedTrigger.EventSource.Schedule.CronExpression != "" {
		state.EventSourceScheduleCronExpression = types.StringValue(updatedTrigger.EventSource.Schedule.CronExpression)
	} else {
		state.EventSourceScheduleCronExpression = types.StringNull()
	}
	// Preserve plan value for schedule triggers; API may transform aliases like "system" → UUID.
	if state.EventSourceProvider.ValueString() != "schedule" {
		if updatedTrigger.EventSource.Schedule.AttributionActor.Id != "" {
			state.EventSourceScheduleAttributionActor = types.StringValue(updatedTrigger.EventSource.Schedule.AttributionActor.Id)
		} else {
			state.EventSourceScheduleAttributionActor = types.StringNull()
		}
	}
	if updatedTrigger.EventPreset == "" {
		state.EventPreset = types.StringNull()
	} else {
		state.EventPreset = types.StringValue(updatedTrigger.EventPreset)
	}
	if updatedTrigger.EventName == "" {
		state.EventName = types.StringNull()
	} else {
		state.EventName = types.StringValue(updatedTrigger.EventName)
	}
	state.CreatedAt = types.StringValue(updatedTrigger.CreatedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *triggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state triggerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.Delete(ctx, state.ProjectId.ValueString(), state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi trigger",
			"Could not delete trigger, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *triggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *circleciClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client.TriggerService
}

func (r *triggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: "PROJECT_ID/TRIGGER_ID"
	parts := strings.SplitN(req.ID, "/", 2)

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Expected import ID format: 'project_id/trigger_id'. Got: %s", req.ID),
		)
		return
	}

	projectId := parts[0]
	triggerId := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("id"), triggerId,
	)...)

	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("project_id"), projectId,
	)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func isValidEventPreset(eventPreset string) bool {
	switch eventPreset {
	case "all-pushes", "only-tags", "default-branch-pushes", "only-build-prs", "only-open-prs", "only-labeled-prs", "only-merged-prs", "only-ready-for-review-prs", "only-branch-delete", "only-build-pushes-to-non-draft-prs", "only-merged-or-closed-prs":
		return true
	default:
		return false
	}
}

func isApiNotFoundError(err error) bool {
	// This is pseudo-code; replace with your actual API client's error inspection
	if apiErr, ok := err.(interface{ HTTPStatusCode() int }); ok {
		return apiErr.HTTPStatusCode() == 404
	}
	// Alternatively, check the error message string if the status is not exposed
	return strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found")
}
