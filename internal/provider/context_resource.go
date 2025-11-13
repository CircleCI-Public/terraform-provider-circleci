// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	ccicontext "github.com/CircleCI-Public/circleci-sdk-go/context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &contextResource{}
	_ resource.ResourceWithConfigure   = &contextResource{}
	_ resource.ResourceWithImportState = &contextResource{}
)

// contextResourceModel maps the output schema.
type contextResourceModel struct {
	OrganizationId types.String `tfsdk:"organization_id"`
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

// NewContextResource is a helper function to simplify the provider implementation.
func NewContextResource() resource.Resource {
	return &contextResource{}
}

// contextResource is the resource implementation.
type contextResource struct {
	client *ccicontext.ContextService
}

// Metadata returns the resource type name.
func (r *contextResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context"
}

// Schema defines the schema for the resource.
func (r *contextResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "organization_id of the circleci context",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci context",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci context",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'name' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci context",
				Computed:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *contextResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan contextResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new context
	newCciContext, err := r.client.Create(ctx, plan.OrganizationId.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI context",
			"Could not create CircleCI context, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.CreatedAt = types.StringValue(newCciContext.CreatedAt)
	plan.Id = types.StringValue(newCciContext.ID)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *contextResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var contextState contextResourceModel
	diags := req.State.Get(ctx, &contextState)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	if contextState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing context id",
			"Missing context id",
		)
		return
	}

	context, err := r.client.Get(ctx, contextState.Id.ValueString())
	if err != nil {
		// Safely retrieve the ID string for the error message
		// Use .ValueString() only after checking IsNull() if this was the first access,
		// but since we rely on it being set, let's simplify the error message to avoid the panic risk.

		contextID := "unknown ID"
		if !contextState.Id.IsNull() {
			contextID = contextState.Id.ValueString()
		}

		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context with id "+contextID,
			err.Error(),
		)
		return
	}

	// ⚠️ CRITICAL FIX: Handle successful transport but no resource returned (nil context)
	if context == nil {
		// This often happens if the context was deleted just before import,
		// or if the API client returns nil instead of an error for a 404.
		resp.Diagnostics.AddWarning(
			"Context not found during Read",
			fmt.Sprintf("Context ID %s could not be retrieved from CircleCI. Removing from state.", contextState.Id.ValueString()),
		)
		// Mark resource for removal from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to model
	contextState = contextResourceModel{
		Id:             types.StringValue(context.ID),
		Name:           types.StringValue(context.Name),
		CreatedAt:      types.StringValue(context.CreatedAt),
		OrganizationId: contextState.OrganizationId,
	}

	// Set state
	diags = resp.State.Set(ctx, &contextState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *contextResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *contextResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state contextResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.Delete(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi Context",
			"Could not delete context, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *contextResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client.ContextService
}

func (r *contextResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("id"), req.ID,
	)...)

	if resp.Diagnostics.HasError() {
		return
	}
}
