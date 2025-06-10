// Copyright (c) HashiCorp, Inc.
// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/common"
	"github.com/CircleCI-Public/circleci-sdk-go/trigger"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	CheckoutRef               types.String `tfsdk:"checkout_ref"`
	ConfigRef                 types.String `tfsdk:"config_ref"`
	EventSourceProvider       types.String `tfsdk:"event_source_provider"`
	EventSourceRepoFullName   types.String `tfsdk:"event_source_repo_full_name"`
	EventSourceRepoExternalId types.String `tfsdk:"event_source_repo_external_id"`
	EventSourceWebHookUrl     types.String `tfsdk:"event_source_web_hook_url"`
	EventPreset               types.String `tfsdk:"event_preset"`
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
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci trigger",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "description of the circleci trigger",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci trigger",
				Computed:            true,
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
			},
			"event_source_repo_external_id": schema.StringAttribute{
				MarkdownDescription: "event_source_repo_external_id of the circleci trigger",
				Required:            true,
			},
			"event_source_web_hook_url": schema.StringAttribute{
				MarkdownDescription: "event_source_web_hook_url of the circleci trigger",
				Computed:            true,
			},
			"event_preset": schema.StringAttribute{
				MarkdownDescription: "event_preset of the circleci trigger",
				Required:            true,
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

	// New Webhook
	newWebHook := common.Webhook{
		Url: circleCiTerrformTriggerResource.EventSourceWebHookUrl.ValueString(),
	}

	// New Repo
	newRepo := common.Repo{
		FullName:   circleCiTerrformTriggerResource.EventSourceRepoFullName.ValueString(),
		ExternalId: circleCiTerrformTriggerResource.EventSourceRepoExternalId.ValueString(),
	}

	// New EventSource
	newEventSource := common.EventSource{
		Provider: circleCiTerrformTriggerResource.EventSourceProvider.ValueString(),
		Repo:     newRepo,
		Webhook:  newWebHook,
	}

	// New Trigger
	newTrigger := trigger.Trigger{
		Name:        circleCiTerrformTriggerResource.Name.ValueString(),
		Description: circleCiTerrformTriggerResource.Description.ValueString(),
		CheckoutRef: circleCiTerrformTriggerResource.CheckoutRef.ValueString(),
		ConfigRef:   circleCiTerrformTriggerResource.ConfigRef.ValueString(),
		EventSource: newEventSource,
		EventPreset: circleCiTerrformTriggerResource.EventPreset.ValueString(),
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
	circleCiTerrformTriggerResource.Name = types.StringValue(newReturnedTrigger.Name)
	circleCiTerrformTriggerResource.Description = types.StringValue(newReturnedTrigger.Description)
	circleCiTerrformTriggerResource.CreatedAt = types.StringValue(newReturnedTrigger.CreatedAt)
	circleCiTerrformTriggerResource.CheckoutRef = types.StringValue(newReturnedTrigger.CheckoutRef)
	circleCiTerrformTriggerResource.ConfigRef = types.StringValue(newReturnedTrigger.ConfigRef)
	circleCiTerrformTriggerResource.EventSourceProvider = types.StringValue(newReturnedTrigger.EventSource.Provider)
	circleCiTerrformTriggerResource.EventSourceRepoFullName = types.StringValue(newReturnedTrigger.EventSource.Repo.FullName)
	//circleCiTerrformTriggerResource.EventSourceRepoExternalId = types.StringValue(newReturnedTrigger.EventSource.Repo.ExternalId)
	circleCiTerrformTriggerResource.EventSourceWebHookUrl = types.StringValue(newReturnedTrigger.EventSource.Webhook.Url)
	//circleCiTerrformTriggerResource.EventPreset = types.StringValue(newReturnedTrigger.EventPreset)

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
	req.State.Get(ctx, &triggerState)

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
	//triggerState.PipelineId = types.StringValue()
	triggerState.Name = types.StringValue(readTrigger.Name)
	triggerState.Description = types.StringValue(readTrigger.Description)
	triggerState.CreatedAt = types.StringValue(readTrigger.CreatedAt)
	triggerState.CheckoutRef = types.StringValue(readTrigger.CheckoutRef)
	triggerState.ConfigRef = types.StringValue(readTrigger.ConfigRef)
	triggerState.EventSourceProvider = types.StringValue(readTrigger.EventSource.Provider)
	triggerState.EventSourceRepoFullName = types.StringValue(readTrigger.EventSource.Repo.FullName)
	triggerState.EventSourceRepoExternalId = types.StringValue(readTrigger.EventSource.Repo.ExternalId)
	triggerState.EventSourceWebHookUrl = types.StringValue(readTrigger.EventSource.Webhook.Url)
	triggerState.EventPreset = types.StringValue(readTrigger.EventPreset)

	// Set state
	diags := resp.State.Set(ctx, &triggerState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *triggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
