// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-circleci/internal/planmodifiers"
	"terraform-provider-circleci/internal/validators"

	"github.com/CircleCI-Public/circleci-sdk-go/common"
	"github.com/CircleCI-Public/circleci-sdk-go/trigger"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &triggerResource{}
	_ resource.ResourceWithConfigure = &triggerResource{}
)

// triggerResourceModel maps the output schema.
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
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "project_id of the circleci trigger",
				Required:            true,
			},
			"pipeline_id": schema.StringAttribute{
				MarkdownDescription: "pipeline_id of the circleci trigger",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci trigger",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checkout_ref": schema.StringAttribute{
				MarkdownDescription: "checkout_ref of the circleci trigger",
				Required:            true,
			},
			"config_ref": schema.StringAttribute{
				MarkdownDescription: "config_ref of the circleci trigger",
				Required:            true,
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
			"event_preset": schema.StringAttribute{
				MarkdownDescription: "event_preset of the circleci trigger",
				Optional:            true,
			},
			"event_name": schema.StringAttribute{
				MarkdownDescription: "event_name of the circleci trigger",
				Optional:            true,
				Validators: []validator.String{
					validators.NewWebhookEventNameValidator(),
				},
				PlanModifiers: []planmodifier.String{
					planmodifiers.NewIgnoreComputedIfGithubAppModifier(),
				},
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
	case "github_app":
		if !isValidEventPreset(circleCiTerrformTriggerResource.EventPreset.ValueString()) {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with github_app provider has an unexpected event_preset",
			)
			return
		}
		if !circleCiTerrformTriggerResource.EventName.IsNull() {
			resp.Diagnostics.AddError(
				"Error creating CircleCI trigger",
				"CircleCI trigger with github_app provider does not support event_name",
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
	default:
		resp.Diagnostics.AddError(
			"Error creating CircleCI trigger",
			"CircleCI trigger has an unexpected event source provider: should be either github_app or webhook",
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

	// New EventSource
	newEventSource := common.EventSource{
		Provider: circleCiTerrformTriggerResource.EventSourceProvider.ValueString(),
		Repo:     newRepo,
		Webhook:  newWebHook,
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
	circleCiTerrformTriggerResource.CheckoutRef = types.StringValue(newReturnedTrigger.CheckoutRef)
	circleCiTerrformTriggerResource.ConfigRef = types.StringValue(newReturnedTrigger.ConfigRef)
	circleCiTerrformTriggerResource.EventSourceProvider = types.StringValue(newReturnedTrigger.EventSource.Provider)
	tflog.Error(ctx, "DAVID CREATE"+newReturnedTrigger.ID+" DAVID END")
	tflog.Error(ctx, "DAVID CREATE"+newReturnedTrigger.CreatedAt+" DAVID END")
	circleCiTerrformTriggerResource.EventSourceRepoFullName = types.StringValue(newReturnedTrigger.EventSource.Repo.FullName)

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
	circleCiTerrformTriggerResource.Disabled = types.BoolValue(*newReturnedTrigger.Disabled)

	readTrigger, err := r.client.Get(circleCiTerrformTriggerResource.ProjectId.ValueString(), newReturnedTrigger.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed retrieving", err.Error())
		// Cleanup may be required here (e.g., Delete the resource if it failed to settle)
		return
	}
	circleCiTerrformTriggerResource.CreatedAt = types.StringValue(readTrigger.CreatedAt)

	tflog.Error(ctx, "DAVID CREATE READ"+circleCiTerrformTriggerResource.Id.ValueString()+" DAVID END")
	tflog.Error(ctx, "DAVID CREATE READ"+circleCiTerrformTriggerResource.CreatedAt.ValueString()+" DAVID END")

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
	diags := req.State.Get(ctx, &triggerState)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	if triggerState.ProjectId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing project_id",
			"Missing project_id",
		)
		return
	}

	if triggerState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing id",
			"Missing id",
		)
		return
	}

	readTrigger, err := r.client.Get(triggerState.ProjectId.ValueString(), triggerState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI trigger with id "+triggerState.Id.ValueString()+" and project id "+triggerState.ProjectId.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	triggerState.Id = types.StringValue(readTrigger.ID)
	triggerState.CheckoutRef = types.StringValue(readTrigger.CheckoutRef)
	triggerState.ConfigRef = types.StringValue(readTrigger.ConfigRef)
	triggerState.EventSourceProvider = types.StringValue(readTrigger.EventSource.Provider)
	if readTrigger.EventSource.Repo.FullName == "" {
		// If the API returns the empty string (the zero value), explicitly save NULL.
		// This aligns the state with the user's omitted config (or the API's lack of data).
		triggerState.EventSourceRepoFullName = types.StringNull()
	} else {
		// If the API returns a non-empty string, save the known value.
		triggerState.EventSourceRepoFullName = types.StringValue(readTrigger.EventSource.Repo.FullName)
	}
	triggerState.EventSourceRepoExternalId = types.StringValue(readTrigger.EventSource.Repo.ExternalId)
	triggerState.EventSourceWebHookUrl = types.StringValue(readTrigger.EventSource.Webhook.Url)
	triggerState.EventPreset = types.StringValue(readTrigger.EventPreset)
	triggerState.EventName = types.StringValue(readTrigger.EventName)
	triggerState.CreatedAt = types.StringValue(readTrigger.CreatedAt)

	// Set state
	diags = resp.State.Set(ctx, &triggerState)
	resp.Diagnostics.Append(diags...)
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

	// New EventSource
	newEventSource := common.EventSource{
		Provider: state.EventSourceProvider.ValueString(),
		Repo:     newRepo,
		Webhook:  newWebHook,
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

	// update the triger
	updatedTrigger, err := r.client.Update(updates, state.ProjectId.ValueString(), state.Id.ValueString())
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
		// If the API returns the empty string (the zero value), explicitly save NULL.
		// This aligns the state with the user's omitted config (or the API's lack of data).
		state.EventSourceRepoFullName = types.StringNull()
	} else {
		// If the API returns a non-empty string, save the known value.
		state.EventSourceRepoFullName = types.StringValue(updatedTrigger.EventSource.Repo.FullName)
	}
	state.EventSourceRepoExternalId = types.StringValue(updatedTrigger.EventSource.Repo.ExternalId)
	state.EventSourceWebHookUrl = types.StringValue(updatedTrigger.EventSource.Webhook.Url)
	state.EventPreset = types.StringValue(updatedTrigger.EventPreset)
	state.EventName = types.StringValue(updatedTrigger.EventName)
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
	err := r.client.Delete(state.ProjectId.ValueString(), state.Id.ValueString())
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

func isValidEventPreset(eventPreset string) bool {
	switch eventPreset {
	case "all-pushes", "only-tags", "default-branch-pushes", "only-build-prs", "only-open-prs", "only-labeled-prs", "only-merged-prs", "only-ready-for-review-prs", "only-branch-delete", "only-build-pushes-to-non-draft-prs", "only-merged-or-closed-prs":
		return true
	default:
		return false
	}
}
