// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/common"
	"github.com/CircleCI-Public/circleci-sdk-go/webhook"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

// webhookResourceModel maps the resource schema.
type webhookResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	VerifyTls     types.Bool   `tfsdk:"verify_tls"`
	SigningSecret types.String `tfsdk:"signing_secret"`
	ScopeId       types.String `tfsdk:"scope_id"`
	ScopeType     types.String `tfsdk:"scope_type"`
	Events        types.List   `tfsdk:"events"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

// NewWebhookResource is a helper function to simplify the provider implementation.
func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

// webhookResource is the resource implementation.
type webhookResource struct {
	client *webhook.WebhookService
}

// Metadata returns the resource type name.
func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines the schema for the resource.
func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CircleCI webhook. Webhooks allow you to receive notifications when events occur in your CircleCI projects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the webhook.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the webhook.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to which webhook payloads will be sent. Must be a valid HTTPS URL.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^https://[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*(:[0-9]{1,5})?(/.*)?$`),
						"URL must be a valid HTTPS URL with a proper hostname",
					),
				},
			},
			"verify_tls": schema.BoolAttribute{
				MarkdownDescription: "Whether to verify TLS certificates when sending payloads. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"signing_secret": schema.StringAttribute{
				MarkdownDescription: "The secret used to sign webhook payloads.",
				Required:            true,
				Sensitive:           true,
			},
			"scope_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the scope (project) for which the webhook is configured.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_type": schema.StringAttribute{
				MarkdownDescription: "The type of the scope. Currently only 'project' is supported.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"events": schema.ListAttribute{
				MarkdownDescription: "The events that will trigger the webhook. Valid values are: workflow-completed, job-completed.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the webhook was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the webhook was last updated.",
				Computed:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert events list to []string
	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the webhook request
	verifyTls := plan.VerifyTls.ValueBool()
	newWebhook := webhook.Webhook{
		Name:          plan.Name.ValueString(),
		Url:           plan.Url.ValueString(),
		VerifyTls:     &verifyTls,
		SigningSecret: plan.SigningSecret.ValueString(),
		Scope: common.Scope{
			Id:   plan.ScopeId.ValueString(),
			Type: plan.ScopeType.ValueString(),
		},
		Events: events,
	}

	// Create the webhook
	createdWebhook, err := r.client.Create(ctx, newWebhook)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI webhook",
			"Could not create CircleCI webhook, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.Id = types.StringValue(createdWebhook.Id)
	// Note: signing_secret is preserved from plan (user-provided value)
	plan.CreatedAt = types.StringValue(createdWebhook.CreatedAt)
	plan.UpdatedAt = types.StringValue(createdWebhook.UpdatedAt)

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing webhook id",
			"Missing webhook id",
		)
		return
	}

	webhookData, err := r.client.Get(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI webhook with id "+state.Id.ValueString(),
			err.Error(),
		)
		return
	}

	// Handle case where webhook was deleted outside of Terraform
	if webhookData == nil {
		resp.Diagnostics.AddWarning(
			"Webhook not found during Read",
			fmt.Sprintf("Webhook ID %s could not be retrieved from CircleCI. Removing from state.", state.Id.ValueString()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Convert events to types.List
	eventsAttributeValues := make([]attr.Value, len(webhookData.Events))
	for i, event := range webhookData.Events {
		eventsAttributeValues[i] = types.StringValue(event)
	}
	eventsList, diags := types.ListValue(types.StringType, eventsAttributeValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map response to state
	state.Id = types.StringValue(webhookData.Id)
	state.Name = types.StringValue(webhookData.Name)
	state.Url = types.StringValue(webhookData.Url)
	if webhookData.VerifyTls != nil {
		state.VerifyTls = types.BoolValue(*webhookData.VerifyTls)
	}
	state.ScopeId = types.StringValue(webhookData.Scope.Id)
	state.ScopeType = types.StringValue(webhookData.Scope.Type)
	state.Events = eventsList
	// Note: created_at, updated_at, and signing_secret may not be returned by Get, preserve from state
	if webhookData.CreatedAt != "" {
		state.CreatedAt = types.StringValue(webhookData.CreatedAt)
	}
	if webhookData.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(webhookData.UpdatedAt)
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state webhookResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert events list to []string
	var events []string
	diags = plan.Events.ElementsAs(ctx, &events, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the webhook update request
	// Note: Scope cannot be updated
	verifyTls := plan.VerifyTls.ValueBool()
	updateWebhook := webhook.Webhook{
		Name:          plan.Name.ValueString(),
		Url:           plan.Url.ValueString(),
		VerifyTls:     &verifyTls,
		SigningSecret: plan.SigningSecret.ValueString(),
		Events:        events,
	}

	// Update the webhook
	updatedWebhook, err := r.client.Update(ctx, updateWebhook, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating CircleCI webhook",
			"Could not update CircleCI webhook, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.Id = state.Id
	// Note: signing_secret is preserved from plan (user-provided value)
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(updatedWebhook.UpdatedAt)

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCI Webhook",
			"Could not delete webhook, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CircleCiClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client.WebhookService
}

// ImportState imports the resource state.
func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: "SCOPE_ID/WEBHOOK_ID"
	parts := strings.SplitN(req.ID, "/", 2)

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Expected import ID format: 'scope_id/webhook_id'. Got: %s", req.ID),
		)
		return
	}

	scopeID := parts[0]
	webhookID := parts[1]

	// Set the primary key 'id'
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("id"), webhookID,
	)...)

	// Set the scope_id
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("scope_id"), scopeID,
	)...)

	// Set the scope_type to "project" as default (currently only supported type)
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("scope_type"), "project",
	)...)

	if resp.Diagnostics.HasError() {
		return
	}
}
