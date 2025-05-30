// Copyright (c) CircleCI and HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0



package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/env"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &contextEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure = &contextEnvironmentVariableResource{}
)

// contextEnvironmentVariableResourceModel maps the output schema.
type contextEnvironmentVariableResourceModel struct {
	Name      types.String `tfsdk:"name"`
	Value     types.String `tfsdk:"value"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	CreatedAt types.String `tfsdk:"created_at"`
	ContextId types.String `tfsdk:"context_id"`
}

// NewContextEnvironmentVariableResource is a helper function to simplify the provider implementation.
func NewContextEnvironmentVariableResource() resource.Resource {
	return &contextEnvironmentVariableResource{}
}

// contextEnvironmentVariableResource is the resource implementation.
type contextEnvironmentVariableResource struct {
	client *env.EnvService
}

// Metadata returns the resource type name.
func (r *contextEnvironmentVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_environment_variable"
}

// Schema defines the schema for the resource.
func (r *contextEnvironmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci context environment variable",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "value of the circleci context environment variable",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'context_id' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "updated date of the circleci context environment variable",
				Computed:            true,
			},
			"context_id": schema.StringAttribute{
				MarkdownDescription: "context id of the circleci context environment variable",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created date of the circleci context environment variable",
				Computed:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *contextEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan contextEnvironmentVariableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new context
	newContextEnvironmentVariable, err := r.client.Create(plan.ContextId.ValueString(), plan.Value.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI context environment variable",
			"Could not create CircleCI context environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.CreatedAt = types.StringValue(newContextEnvironmentVariable.CreatedAt)
	plan.UpdatedAt = types.StringValue(newContextEnvironmentVariable.UpdatedAt)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *contextEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var contextEnvironmentVariableState contextEnvironmentVariableResourceModel
	req.State.Get(ctx, &contextEnvironmentVariableState)

	if contextEnvironmentVariableState.ContextId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing environment variable context id",
			"Missing environment variable context id",
		)
		return
	}

	if contextEnvironmentVariableState.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Missing environment variable name",
			"Missing environment variable name",
		)
		return
	}

	if contextEnvironmentVariableState.Value.IsNull() {
		resp.Diagnostics.AddError(
			"Missing environment variable value",
			"Missing environment variable value",
		)
		return
	}

	contextEnvironmentVariables, err := r.client.List(contextEnvironmentVariableState.ContextId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context environment variable with context id "+contextEnvironmentVariableState.ContextId.ValueString(),
			err.Error(),
		)
		return
	}

	// Fill restrictions
	for _, elem := range contextEnvironmentVariables {
		if elem.Variable == contextEnvironmentVariableState.Name.ValueString() {
			contextEnvironmentVariableState.Name = types.StringValue(elem.Variable)
			contextEnvironmentVariableState.UpdatedAt = types.StringValue(elem.UpdatedAt)
			contextEnvironmentVariableState.CreatedAt = types.StringValue(elem.CreatedAt)
			contextEnvironmentVariableState.ContextId = types.StringValue(elem.ContextId)
			break
		}
	}

	// Set state
	diags := resp.State.Set(ctx, &contextEnvironmentVariableState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *contextEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *contextEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state contextEnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.Delete(state.ContextId.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi Context Environment Variable",
			"Could not delete context, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *contextEnvironmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client.EnvironmentVariableService
}
