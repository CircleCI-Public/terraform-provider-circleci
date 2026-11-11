// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/group"
)

var (
	_ resource.Resource                = &groupProjectRoleResource{}
	_ resource.ResourceWithConfigure   = &groupProjectRoleResource{}
	_ resource.ResourceWithImportState = &groupProjectRoleResource{}
)

type groupProjectRoleResourceModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	GroupId        types.String `tfsdk:"group_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Role           types.String `tfsdk:"role"`
}

// NewGroupProjectRoleResource returns the resource for the role one group
// holds on one project.
func NewGroupProjectRoleResource() resource.Resource {
	return &groupProjectRoleResource{}
}

type groupProjectRoleResource struct {
	client *group.Service
}

func (r *groupProjectRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_project_role"
}

func (r *groupProjectRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants a CircleCI group a role on a project. Only standalone (`circleci`) organizations support reading the grant back.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the grant, of the form `group_id/project_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns the group and the project. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "The role to grant. Must be one of `project-admin`, `project-contributor` or `project-viewer`. Changing this value forces a new resource, so the group briefly holds no role on the project.",
				Required:            true,
				PlanModifiers:       replace,
				Validators:          []validator.String{stringvalidator.OneOf(group.ProjectRoles...)},
			},
		},
	}
}

func (r *groupProjectRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupProjectRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID, projectID := plan.GroupId.ValueString(), plan.ProjectId.ValueString()
	err := r.client.GrantProjects(ctx, plan.OrganizationId.ValueString(), groupID, plan.Role.ValueString(), projectID)
	if err != nil {
		resp.Diagnostics.AddError("Error granting CircleCI group "+groupID+" a role on project "+projectID, err.Error())
		return
	}
	plan.Id = types.StringValue(groupID + "/" + projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *groupProjectRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupProjectRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := r.client.ProjectGroups(ctx, state.OrganizationId.ValueString(), state.ProjectId.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading the groups of CircleCI project "+state.ProjectId.ValueString(), err.Error())
		return
	}
	i := slices.IndexFunc(groups, func(g group.ProjectGroup) bool { return g.ID == state.GroupId.ValueString() })
	if i < 0 {
		resp.State.RemoveResource(ctx)
		return
	}
	state.Id = types.StringValue(state.GroupId.ValueString() + "/" + state.ProjectId.ValueString())
	state.Role = types.StringValue(groups[i].Role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupProjectRoleResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "Every attribute of circleci_group_project_role forces a new resource. Please report this issue to the provider developers.")
}

func (r *groupProjectRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupProjectRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RevokeProjects(ctx, state.OrganizationId.ValueString(), state.GroupId.ValueString(), state.ProjectId.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error revoking the role of CircleCI group "+state.GroupId.ValueString()+" on project "+state.ProjectId.ValueString(), err.Error())
	}
}

func (r *groupProjectRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureGroupService(req, resp)
}

func (r *groupProjectRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 3)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected an import ID of the form organization_id/group_id/project_id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[2])...)
}
