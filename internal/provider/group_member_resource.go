// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/group"
)

var (
	_ resource.Resource                = &groupMemberResource{}
	_ resource.ResourceWithConfigure   = &groupMemberResource{}
	_ resource.ResourceWithImportState = &groupMemberResource{}
)

type groupMemberResourceModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	GroupId        types.String `tfsdk:"group_id"`
	UserId         types.String `tfsdk:"user_id"`
	Username       types.String `tfsdk:"username"`
}

// NewGroupMemberResource returns the resource for one user's membership of a
// group.
func NewGroupMemberResource() resource.Resource {
	return &groupMemberResource{}
}

type groupMemberResource struct {
	client *group.Service
}

func (r *groupMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_member"
}

func (r *groupMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Adds a user to a CircleCI group. Each resource manages one membership, so members added outside Terraform are left alone. " +
			"The user must already be a member of the organization; CircleCI has no API to invite users or to look them up by name, so `user_id` is the user's CircleCI ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the membership, of the form `group_id/user_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns the group. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the group. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The CircleCI ID of the user. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       replace,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The user's username, when CircleCI reports it.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *groupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID, groupID, userID := plan.OrganizationId.ValueString(), plan.GroupId.ValueString(), plan.UserId.ValueString()
	if err := r.client.AddMembers(ctx, orgID, groupID, userID); err != nil {
		resp.Diagnostics.AddError("Error adding user "+userID+" to CircleCI group "+groupID, err.Error())
		return
	}
	member, found, err := r.find(ctx, orgID, groupID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the members of CircleCI group "+groupID, err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("User not added to CircleCI group",
			fmt.Sprintf("CircleCI accepted user %s for group %s but does not list them as a member. Check that the user belongs to the organization.", userID, groupID))
		return
	}

	plan.Id = types.StringValue(groupID + "/" + userID)
	plan.Username = types.StringValue(member.Username)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *groupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, found, err := r.find(ctx, state.OrganizationId.ValueString(), state.GroupId.ValueString(), state.UserId.ValueString())
	if client.IsNotFound(err) || (err == nil && !found) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading the members of CircleCI group "+state.GroupId.ValueString(), err.Error())
		return
	}
	state.Id = types.StringValue(state.GroupId.ValueString() + "/" + state.UserId.ValueString())
	state.Username = types.StringValue(member.Username)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupMemberResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "Every attribute of circleci_group_member forces a new resource. Please report this issue to the provider developers.")
}

func (r *groupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RemoveMembers(ctx, state.OrganizationId.ValueString(), state.GroupId.ValueString(), state.UserId.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error removing user "+state.UserId.ValueString()+" from CircleCI group "+state.GroupId.ValueString(), err.Error())
	}
}

func (r *groupMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureGroupService(req, resp)
}

func (r *groupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 3)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected an import ID of the form organization_id/group_id/user_id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[2])...)
}

func (r *groupMemberResource) find(ctx context.Context, orgID, groupID, userID string) (group.Member, bool, error) {
	members, err := r.client.Members(ctx, orgID, groupID)
	if err != nil {
		return group.Member{}, false, err
	}
	i := slices.IndexFunc(members, func(m group.Member) bool { return m.UserID == userID })
	if i < 0 {
		return group.Member{}, false, nil
	}
	return members[i], true, nil
}
