// Copyright (c) HashiCorp, Inc.
// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	ccicontext "github.com/CircleCI-Public/circleci-sdk-go/context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &contextRestrictionResource{}
	_ resource.ResourceWithConfigure = &contextRestrictionResource{}
)

// contextResourceModel maps the output schema.
type contextRestrictionResourceModel struct {
	Id        types.String `tfsdk:"id"`
	ContextId types.String `tfsdk:"context_id"`
	ProjectId types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Value     types.String `tfsdk:"value"`
}

// NewContextResource is a helper function to simplify the provider implementation.
func NewContextRestrictionResource() resource.Resource {
	return &contextRestrictionResource{}
}

// contextResource is the resource implementation.
type contextRestrictionResource struct {
	client *ccicontext.ContextService
}

// Metadata returns the resource type name.
func (r *contextRestrictionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_restriction"
}

// Schema defines the schema for the resource.
func (r *contextRestrictionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci context restriction",
				Computed:            true,
			},
			"context_id": schema.StringAttribute{
				MarkdownDescription: "context_id of the circleci context restriction",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'context_id' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "project_id of the circleci context restriction",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci context restriction",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "type of the circleci context restriction",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'type' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "value of the circleci context restriction",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'value' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *contextRestrictionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan contextRestrictionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new context
	tflog.Info(ctx, fmt.Sprintf("Create Restriction: %+v", plan))
	newCciContextRestriction, err := r.client.CreateRestriction(plan.ContextId.ValueString(), plan.Value.ValueString(), plan.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI context",
			"Could not create CircleCI context restriction, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringValue(newCciContextRestriction.ID)
	plan.ProjectId = types.StringValue(newCciContextRestriction.ProjectId)
	plan.Name = types.StringValue(newCciContextRestriction.Name)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *contextRestrictionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var contextRestrictionState contextRestrictionResourceModel
	req.State.Get(ctx, &contextRestrictionState)

	if contextRestrictionState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing context id",
			"Missing context id",
		)
		return
	}

	restrictions, err := r.client.GetRestrictions(contextRestrictionState.ContextId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context restriction with id "+contextRestrictionState.Id.ValueString(),
			err.Error(),
		)
		return
	}

	var cciContextRestriction ccicontext.ContextRestriction
	for _, restriction := range restrictions {
		if restriction.ID == contextRestrictionState.Id.ValueString() {
			cciContextRestriction = restriction
			break
		}
	}

	// Map response body to model
	contextRestrictionState = contextRestrictionResourceModel{
		Id:        types.StringValue(cciContextRestriction.ID),
		ContextId: types.StringValue(cciContextRestriction.ContextId),
		ProjectId: types.StringValue(cciContextRestriction.ProjectId),
		Name:      types.StringValue(cciContextRestriction.Name),
		Type:      types.StringValue(cciContextRestriction.RestrictionType),
		Value:     types.StringValue(cciContextRestriction.RestrictionValue),
	}

	// Set state
	diags := resp.State.Set(ctx, &contextRestrictionState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *contextRestrictionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *contextRestrictionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state contextRestrictionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.DeleteRestriction(state.ContextId.ValueString(), state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi Context",
			"Could not delete context, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *contextRestrictionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
