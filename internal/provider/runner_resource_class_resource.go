// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/runner"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &runnerResourceClassResource{}
	_ resource.ResourceWithConfigure   = &runnerResourceClassResource{}
	_ resource.ResourceWithImportState = &runnerResourceClassResource{}
)

// runnerResourceClassResourceModel maps the resource schema.
type runnerResourceClassResourceModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	ResourceClass  types.String `tfsdk:"resource_class"`
	Description    types.String `tfsdk:"description"`
	ForceDelete    types.Bool   `tfsdk:"force_delete"`
}

// NewRunnerResourceClassResource is a helper function to simplify the provider implementation.
func NewRunnerResourceClassResource() resource.Resource {
	return &runnerResourceClassResource{}
}

// runnerResourceClassResource is the resource implementation.
type runnerResourceClassResource struct {
	client *runner.Service
}

// Metadata returns the resource type name.
func (r *runnerResourceClassResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_resource_class"
}

// Schema defines the schema for the resource.
func (r *runnerResourceClassResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CircleCI runner resource class.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier (UUID) of the runner resource class.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The organization id.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_class": schema.StringAttribute{
				MarkdownDescription: "The resource class name in `namespace/name` format (e.g. `myorg/myrunner`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the runner resource class.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"force_delete": schema.BoolAttribute{
				MarkdownDescription: "If true, deletes the resource class even if it has associated tokens.",
				Optional:            true,
				Computed:            false,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *runnerResourceClassResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan runnerResourceClassResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := runner.CreateResourceClassRequest{
		ResourceClass: plan.ResourceClass.ValueString(),
		Description:   plan.Description.ValueString(),
	}

	rc, err := r.client.CreateResourceClass(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI runner resource class",
			"Could not create runner resource class, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(rc.Id)
	plan.ResourceClass = types.StringValue(rc.ResourceClass)
	plan.Description = types.StringValue(rc.Description)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *runnerResourceClassResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state runnerResourceClassResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rcName := state.ResourceClass.ValueString()
	slashIdx := strings.Index(rcName, "/")
	if slashIdx == -1 {
		resp.Diagnostics.AddError(
			"Invalid resource_class format",
			fmt.Sprintf("Expected namespace/name format, got: %s", rcName),
		)
		return
	}
	namespace := rcName[:slashIdx]

	classes, err := r.client.ListResourceClasses(ctx, namespace, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading CircleCI runner resource classes",
			"Could not list runner resource classes for namespace "+namespace+": "+err.Error(),
		)
		return
	}

	var found *runner.ResourceClass
	for i := range classes.Items {
		if classes.Items[i].ResourceClass == rcName {
			found = &classes.Items[i]
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Id = types.StringValue(found.Id)
	state.ResourceClass = types.StringValue(found.ResourceClass)
	state.Description = types.StringValue(found.Description)
	// ForceDelete is not returned by the API — preserve value from state.

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource. All mutable fields require replacement, so this is a no-op.
func (r *runnerResourceClassResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *runnerResourceClassResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state runnerResourceClassResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteResourceClass(ctx, state.Id.ValueString(), state.ForceDelete.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting CircleCI runner resource class",
			"Could not delete runner resource class "+state.Id.ValueString()+": "+err.Error(),
		)
	}
}

// Configure adds the provider configured client to the resource.
func (r *runnerResourceClassResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client.RunnerService
}

// ImportState imports an existing resource class into Terraform state.
// The import ID is the resource_class string (e.g. "myorg/myrunner").
// After import, Read is called to populate the full state.
func (r *runnerResourceClassResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_class"), req.ID)...)
}
