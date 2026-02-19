// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/envproject"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure   = &projectEnvironmentVariableResource{}
	_ resource.ResourceWithImportState = &projectEnvironmentVariableResource{}
)

// projectEnvironmentVariableResourceModel maps the resource schema.
type projectEnvironmentVariableResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	ProjectSlug types.String `tfsdk:"project_slug"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

// NewProjectEnvironmentVariableResource is a helper function to simplify the provider implementation.
func NewProjectEnvironmentVariableResource() resource.Resource {
	return &projectEnvironmentVariableResource{}
}

// projectEnvironmentVariableResource is the resource implementation.
type projectEnvironmentVariableResource struct {
	client *envproject.EnvService
}

// Metadata returns the resource type name.
func (r *projectEnvironmentVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variable"
}

// Schema defines the schema for the resource.
func (r *projectEnvironmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci project environment variable",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "value of the circleci project environment variable",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_slug": schema.StringAttribute{
				MarkdownDescription: "project slug of the circleci project environment variable (e.g. circleci/org/project)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created date of the circleci project environment variable",
				Computed:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectEnvironmentVariableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new project environment variable
	newEnvVar, err := r.client.Create(ctx, plan.ProjectSlug.ValueString(), plan.Value.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI project environment variable",
			"Could not create CircleCI project environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	if !newEnvVar.CreatedAt.IsZero() {
		plan.CreatedAt = types.StringValue(newEnvVar.CreatedAt.Format("2006-01-02T15:04:05.000Z"))
	} else {
		plan.CreatedAt = types.StringValue("")
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectEnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	envVar, err := r.client.Get(ctx, state.ProjectSlug.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(envVar.Name)
	// Preserve Value from state since the API returns masked values (e.g. xxxx1234)
	if !envVar.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(envVar.CreatedAt.Format("2006-01-02T15:04:05.000Z"))
	} else {
		state.CreatedAt = types.StringValue("")
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state projectEnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing project environment variable
	err := r.client.Delete(ctx, state.ProjectSlug.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi Project Environment Variable",
			"Could not delete project environment variable, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectEnvironmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client.ProjectEnvironmentVariableService
}

// ImportState imports an existing resource into Terraform state.
// Expected import ID format: "project_slug/env_var_name".
// e.g. "circleci/org_id/project_id/MY_VAR".
func (r *projectEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The project slug contains slashes (e.g. "circleci/org/project"),
	// so split from the right to extract the env var name.
	lastSlash := strings.LastIndex(req.ID, "/")
	if lastSlash == -1 || lastSlash == 0 || lastSlash == len(req.ID)-1 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Expected import ID format: 'project_slug/env_var_name' (e.g. 'circleci/org_id/project_id/MY_VAR'). Got: %s", req.ID),
		)
		return
	}

	projectSlug := req.ID[:lastSlash]
	name := req.ID[lastSlash+1:]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}
