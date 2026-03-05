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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &runnerTokenResource{}
	_ resource.ResourceWithConfigure   = &runnerTokenResource{}
	_ resource.ResourceWithImportState = &runnerTokenResource{}
)

// runnerTokenResourceModel maps the resource schema.
type runnerTokenResourceModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	ResourceClass  types.String `tfsdk:"resource_class"`
	Nickname       types.String `tfsdk:"nickname"`
	Token          types.String `tfsdk:"token"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

// NewRunnerTokenResource is a helper function to simplify the provider implementation.
func NewRunnerTokenResource() resource.Resource {
	return &runnerTokenResource{}
}

// runnerTokenResource is the resource implementation.
type runnerTokenResource struct {
	client *runner.Service
}

// Metadata returns the resource type name.
func (r *runnerTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_token"
}

// Schema defines the schema for the resource.
func (r *runnerTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CircleCI runner authentication token. The token value is only available at creation time and cannot be retrieved afterwards.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier (UUID) of the runner token.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The organization id of theresource class this token as a UUID string.",
				Required:            true,
				Computed:            false,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_class": schema.StringAttribute{
				MarkdownDescription: "The resource class this token grants access to, in `namespace/name` format (e.g. `myorg/myrunner`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nickname": schema.StringAttribute{
				MarkdownDescription: "A human-readable label for the token.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token value used to authenticate a runner agent. Only available at creation time — this value is not returned by the API on subsequent reads and will be empty after an import.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time at which the token was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *runnerTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan runnerTokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := runner.CreateTokenRequest{
		OrganizationID: plan.OrganizationId.ValueString(),
		ResourceClass:  plan.ResourceClass.ValueString(),
		Nickname:       plan.Nickname.ValueString(),
	}

	t, err := r.client.CreateToken(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI runner token",
			"Could not create runner token, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(t.Id)
	plan.ResourceClass = types.StringValue(t.ResourceClass)
	plan.Nickname = types.StringValue(t.Nickname)
	plan.Token = types.StringValue(t.Token)
	plan.CreatedAt = types.StringValue(t.CreatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *runnerTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state runnerTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokens, err := r.client.ListTokens(ctx, state.ResourceClass.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading CircleCI runner tokens",
			"Could not list runner tokens for resource class "+state.ResourceClass.ValueString()+": "+err.Error(),
		)
		return
	}

	var found *runner.Token
	for i := range tokens {
		if tokens[i].Id == state.Id.ValueString() {
			found = &tokens[i]
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Id = types.StringValue(found.Id)
	state.ResourceClass = types.StringValue(found.ResourceClass)
	state.Nickname = types.StringValue(found.Nickname)
	state.CreatedAt = types.StringValue(found.CreatedAt)
	// Token is write-once and not returned by the API — preserve value from state.

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource. All fields require replacement, so this is a no-op.
func (r *runnerTokenResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *runnerTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state runnerTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteToken(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting CircleCI runner token",
			"Could not delete runner token "+state.Id.ValueString()+": "+err.Error(),
		)
	}
}

// Configure adds the provider configured client to the resource.
func (r *runnerTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ImportState imports an existing runner token into Terraform state.
// The import ID format is "resource_class/token_id" (e.g. "myorg/myrunner/550e8400-...").
// Note: the token value cannot be recovered after import.
func (r *runnerTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// resource_class is "namespace/name" (one slash), token_id is a UUID (no slashes).
	// Split on the last slash to separate them.
	lastSlash := strings.LastIndex(req.ID, "/")
	if lastSlash == -1 || lastSlash == 0 || lastSlash == len(req.ID)-1 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Expected format: resource_class/token_id (e.g. myorg/myrunner/550e8400-...). Got: %s", req.ID),
		)
		return
	}

	resourceClass := req.ID[:lastSlash]
	tokenID := req.ID[lastSlash+1:]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_class"), resourceClass)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), tokenID)...)
}
