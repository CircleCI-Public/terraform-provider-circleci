// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/notification"
)

var (
	_ datasource.DataSource              = &notificationPreferencesDataSource{}
	_ datasource.DataSourceWithConfigure = &notificationPreferencesDataSource{}
)

type notificationPreferencesDataSourceModel struct {
	OrganizationId types.String                  `tfsdk:"organization_id"`
	ProjectId      types.String                  `tfsdk:"project_id"`
	Preferences    []notificationPreferenceModel `tfsdk:"preferences"`
}

type notificationPreferenceModel struct {
	Id                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Channel            types.String `tfsdk:"channel"`
	GroupName          types.String `tfsdk:"group_name"`
	SectionName        types.String `tfsdk:"section_name"`
	IsUserConfigurable types.Bool   `tfsdk:"is_user_configurable"`
	IsEnabled          types.Bool   `tfsdk:"is_enabled"`
}

// NewNotificationPreferencesDataSource returns the data source that lists a
// project's notification preferences, so their IDs can be looked up by name.
func NewNotificationPreferencesDataSource() datasource.DataSource {
	return &notificationPreferencesDataSource{}
}

type notificationPreferencesDataSource struct {
	client *notification.Service
}

func (d *notificationPreferencesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_preferences"
}

func (d *notificationPreferencesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the notification preferences of a CircleCI project, in the order CircleCI displays them, " +
			"so that their IDs can be looked up by name for `circleci_project_notification_preferences`.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns the project.",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Required:            true,
			},
			"preferences": schema.ListNestedAttribute{
				MarkdownDescription: "The project's notification preferences.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the preference. It is the same for every project.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the preference, such as `Job awaiting approval`.",
							Computed:            true,
						},
						"channel": schema.StringAttribute{
							MarkdownDescription: "The channel the preference applies to, such as `slack`.",
							Computed:            true,
						},
						"group_name": schema.StringAttribute{
							MarkdownDescription: "The group the preference is shown under, such as `Workflow Completed`.",
							Computed:            true,
						},
						"section_name": schema.StringAttribute{
							MarkdownDescription: "The section the preference is shown under, such as `Build`.",
							Computed:            true,
						},
						"is_user_configurable": schema.BoolAttribute{
							MarkdownDescription: "Whether the preference can be turned on or off. Only these can be used in `circleci_project_notification_preferences`.",
							Computed:            true,
						},
						"is_enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether the preference is on for the project.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *notificationPreferencesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state notificationPreferencesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	prefs, err := d.client.ListProjectPreferences(ctx, state.OrganizationId.ValueString(), state.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing CircleCI notification preferences of project "+state.ProjectId.ValueString(), err.Error())
		return
	}
	state.Preferences = make([]notificationPreferenceModel, 0, len(prefs))
	for _, p := range prefs {
		state.Preferences = append(state.Preferences, notificationPreferenceModel{
			Id:                 types.StringValue(p.ID),
			Name:               types.StringValue(p.Name),
			Channel:            types.StringValue(p.Channel),
			GroupName:          types.StringValue(p.GroupName),
			SectionName:        types.StringValue(p.SectionName),
			IsUserConfigurable: types.BoolValue(p.UserConfigurable),
			IsEnabled:          types.BoolValue(p.Enabled),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *notificationPreferencesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	wrapper, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *CircleCiClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = wrapper.NotificationService
}
