// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/organization"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &organizationResource{}
	_ resource.ResourceWithConfigure   = &organizationResource{}
	_ resource.ResourceWithImportState = &organizationResource{}
)

// organizationResourceModel maps the resource schema.
type organizationResourceModel struct {
	Id      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Slug    types.String `tfsdk:"slug"`
	VcsType types.String `tfsdk:"vcs_type"`
}

// NewOrganizationResource is a helper function to simplify the provider implementation.
func NewOrganizationResource() resource.Resource {
	return &organizationResource{}
}

// organizationResource is the resource implementation.
type organizationResource struct {
	client *organization.OrganizationService
}

// Metadata returns the resource type name.
func (r *organizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the resource.
func (r *organizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CircleCI organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID of the CircleCI organization.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the CircleCI organization.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the CircleCI organization.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_type": schema.StringAttribute{
				MarkdownDescription: "The VCS type of the CircleCI organization (e.g., github, bitbucket, circleci).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.Create(ctx, plan.Name.ValueString(), plan.VcsType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI organization",
			"Could not create CircleCI organization, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringValue(org.Id)
	plan.Slug = types.StringValue(org.Slug)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationResourceModel
	diags := req.State.Get(ctx, &state)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	if state.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing organization id",
			"Missing organization id",
		)
		return
	}

	org, err := r.client.Get(ctx, state.Id.ValueString())
	if err != nil {
		orgID := "unknown ID"
		if !state.Id.IsNull() {
			orgID = state.Id.ValueString()
		}

		resp.Diagnostics.AddError(
			"Unable to Read CircleCI organization with id "+orgID,
			err.Error(),
		)
		return
	}

	if org == nil {
		resp.Diagnostics.AddWarning(
			"Organization not found during Read",
			fmt.Sprintf("Organization ID %s could not be retrieved from CircleCI. Removing from state.", state.Id.ValueString()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response body to model
	state = organizationResourceModel{
		Id:      types.StringValue(org.Id),
		Name:    types.StringValue(org.Name),
		Slug:    types.StringValue(org.Slug),
		VcsType: types.StringValue(org.VcsType),
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Organizations cannot be updated - name and vcs_type changes require replacement
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCI Organization",
			"Could not delete organization, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *organizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client.OrganizationService
}

// ImportState imports an existing resource into Terraform state.
func (r *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the organization ID
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("id"), req.ID,
	)...)
}
