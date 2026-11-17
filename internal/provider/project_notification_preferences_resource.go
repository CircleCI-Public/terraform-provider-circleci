// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/notification"
)

var (
	_ resource.Resource                = &projectNotificationPreferencesResource{}
	_ resource.ResourceWithConfigure   = &projectNotificationPreferencesResource{}
	_ resource.ResourceWithImportState = &projectNotificationPreferencesResource{}
)

type projectNotificationPreferencesResourceModel struct {
	Id                   types.String `tfsdk:"id"`
	OrganizationId       types.String `tfsdk:"organization_id"`
	ProjectId            types.String `tfsdk:"project_id"`
	EnabledPreferenceIds types.Set    `tfsdk:"enabled_preference_ids"`
}

// NewProjectNotificationPreferencesResource returns the resource that
// authoritatively sets which notifications a project sends.
func NewProjectNotificationPreferencesResource() resource.Resource {
	return &projectNotificationPreferencesResource{}
}

type projectNotificationPreferencesResource struct {
	client *notification.Service
}

func (r *projectNotificationPreferencesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_notification_preferences"
}

func (r *projectNotificationPreferencesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sets which notifications a CircleCI project sends to its Slack channel, as configured under " +
			"Project Settings > Slack Notifications. Use at most one per project.\n\n" +
			"This resource is authoritative: every user-configurable preference that is not in " +
			"`enabled_preference_ids` is turned off, including ones turned on outside Terraform. " +
			"Destroying it turns every user-configurable preference off. " +
			"Look preference IDs up with the `circleci_notification_preferences` data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`<organization_id>/<project_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns the project. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. Changing this value forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled_preference_ids": schema.SetAttribute{
				MarkdownDescription: "The IDs of the preferences to turn on. Every other user-configurable preference is turned off.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *projectNotificationPreferencesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectNotificationPreferencesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *projectNotificationPreferencesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectNotificationPreferencesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// apply turns on exactly the preferences in plan and every other
// user-configurable one off, in a single bulk update, then reads them back.
func (r *projectNotificationPreferencesResource) apply(ctx context.Context, plan *projectNotificationPreferencesResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	orgID, projectID := plan.OrganizationId.ValueString(), plan.ProjectId.ValueString()

	var want []string
	diags.Append(plan.EnabledPreferenceIds.ElementsAs(ctx, &want, false)...)
	if diags.HasError() {
		return diags
	}

	prefs, err := r.client.ListProjectPreferences(ctx, orgID, projectID)
	if err != nil {
		diags.AddError("Error listing CircleCI notification preferences of project "+projectID, err.Error())
		return diags
	}

	updates := make(map[string]bool, len(prefs))
	known := make(map[string]notification.Preference, len(prefs))
	for _, p := range prefs {
		known[p.ID] = p
		if p.UserConfigurable {
			updates[p.ID] = slices.Contains(want, p.ID)
		}
	}
	for _, id := range want {
		p, ok := known[id]
		switch {
		case !ok:
			diags.AddAttributeError(path.Root("enabled_preference_ids"), "Unknown notification preference",
				fmt.Sprintf("%q is not a notification preference of project %s. Valid IDs are: %s. "+
					"The circleci_notification_preferences data source lists them with their names.",
					id, projectID, configurableIDs(prefs)))
		case !p.UserConfigurable:
			diags.AddAttributeError(path.Root("enabled_preference_ids"), "Notification preference not configurable",
				fmt.Sprintf("Preference %q (%s) cannot be configured by users.", id, p.Name))
		}
	}
	if diags.HasError() {
		return diags
	}

	if err := r.client.UpdateProjectPreferences(ctx, orgID, projectID, updates); err != nil {
		diags.AddError("Error updating CircleCI notification preferences of project "+projectID, err.Error())
		return diags
	}

	plan.Id = types.StringValue(orgID + "/" + projectID)
	d, err := r.readInto(ctx, plan)
	if err != nil {
		diags.AddError("Error reading CircleCI notification preferences of project "+projectID, err.Error())
		return diags
	}
	diags.Append(d...)
	return diags
}

func configurableIDs(prefs []notification.Preference) string {
	ids := make([]string, 0, len(prefs))
	for _, p := range prefs {
		if p.UserConfigurable {
			ids = append(ids, fmt.Sprintf("%s (%s / %s / %s)", p.ID, p.SectionName, p.GroupName, p.Name))
		}
	}
	return strings.Join(ids, ", ")
}

// readInto lists the project's preferences and sets enabled_preference_ids
// from the user-configurable ones that are on. It returns the list error
// as is, so a caller can tell a missing project apart.
func (r *projectNotificationPreferencesResource) readInto(ctx context.Context, m *projectNotificationPreferencesResourceModel) (diag.Diagnostics, error) {
	prefs, err := r.client.ListProjectPreferences(ctx, m.OrganizationId.ValueString(), m.ProjectId.ValueString())
	if err != nil {
		return nil, err
	}
	enabled := make([]string, 0, len(prefs))
	for _, p := range prefs {
		if p.UserConfigurable && p.Enabled {
			enabled = append(enabled, p.ID)
		}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, enabled)
	m.EnabledPreferenceIds = set
	return diags, nil
}

func (r *projectNotificationPreferencesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectNotificationPreferencesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	d, err := r.readInto(ctx, &state)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading CircleCI notification preferences of project "+state.ProjectId.ValueString(), err.Error())
		return
	}
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete turns every user-configurable preference off, since preferences
// cannot be removed and there is no reliable default to restore.
func (r *projectNotificationPreferencesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectNotificationPreferencesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	orgID, projectID := state.OrganizationId.ValueString(), state.ProjectId.ValueString()

	prefs, err := r.client.ListProjectPreferences(ctx, orgID, projectID)
	if client.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error listing CircleCI notification preferences of project "+projectID, err.Error())
		return
	}
	updates := map[string]bool{}
	for _, p := range prefs {
		if p.UserConfigurable {
			updates[p.ID] = false
		}
	}
	err = r.client.UpdateProjectPreferences(ctx, orgID, projectID, updates)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error turning off CircleCI notification preferences of project "+projectID, err.Error())
	}
}

func (r *projectNotificationPreferencesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureNotificationService(req, resp)
}

// ImportState accepts "<organization_id>/<project_id>".
func (r *projectNotificationPreferencesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 2)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected an import ID of the form organization_id/project_id, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[1])...)
}
