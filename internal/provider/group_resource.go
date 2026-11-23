// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/group"
)

var (
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

type groupResourceModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
}

// NewGroupResource returns the resource for an organization's RBAC group,
// which is how roles are managed in an organization without IdP role sync.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client *group.Service
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a group in a CircleCI organization. Add users to it with `circleci_group_member`, and grant it a role on a project with `circleci_group_project_role`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns the group. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the group.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the group. Defaults to an empty string.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
		},
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	g, err := r.client.Create(ctx, plan.OrganizationId.ValueString(), plan.Name.ValueString(), plan.Description.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating CircleCI group", err.Error())
		return
	}
	plan.Id = types.StringValue(g.ID)
	plan.Name = types.StringValue(g.Name)
	plan.Description = types.StringValue(g.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	g, err := r.client.Get(ctx, state.OrganizationId.ValueString(), state.Id.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading CircleCI group "+state.Id.ValueString(), err.Error())
		return
	}
	state.Name = types.StringValue(g.Name)
	state.Description = types.StringValue(g.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	g, err := r.client.Update(ctx, plan.OrganizationId.ValueString(), plan.Id.ValueString(), plan.Name.ValueString(), plan.Description.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating CircleCI group "+plan.Id.ValueString(), err.Error())
		return
	}
	plan.Name = types.StringValue(g.Name)
	plan.Description = types.StringValue(g.Description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, state.OrganizationId.ValueString(), state.Id.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting CircleCI group "+state.Id.ValueString(), err.Error())
	}
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureGroupService(req, resp)
}

// ImportState accepts "<organization_id>/<group_id>", because the API needs
// the organization to look a group up.
func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 2)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected an import ID of the form organization_id/group_id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func configureGroupService(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *group.Service {
	if req.ProviderData == nil {
		return nil
	}
	wrapper, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CircleCiClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}
	return wrapper.GroupService
}

// splitImportID splits a "/"-separated import ID into exactly n non-empty
// parts.
func splitImportID(id string, n int) ([]string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != n {
		return nil, false
	}
	for _, p := range parts {
		if p == "" {
			return nil, false
		}
	}
	return parts, true
}
