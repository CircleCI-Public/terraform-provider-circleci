// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/notification"
)

var (
	_ resource.Resource                = &projectSlackNotificationResource{}
	_ resource.ResourceWithConfigure   = &projectSlackNotificationResource{}
	_ resource.ResourceWithImportState = &projectSlackNotificationResource{}
)

// slackChannelIDPattern matches a Slack channel ID such as C0123456789. The
// API reports the channel as an ID, so a channel name would never match what
// is read back and would show as a change on every plan.
var slackChannelIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{5,}$`)

type projectSlackNotificationResourceModel struct {
	Id             types.String `tfsdk:"id"`
	OrganizationId types.String `tfsdk:"organization_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	Channel        types.String `tfsdk:"channel"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	ChannelName    types.String `tfsdk:"channel_name"`
	DisableReason  types.String `tfsdk:"disable_reason"`
}

// NewProjectSlackNotificationResource returns the resource for the Slack
// channel a project's notifications are posted to.
func NewProjectSlackNotificationResource() resource.Resource {
	return &projectSlackNotificationResource{}
}

type projectSlackNotificationResource struct {
	client *notification.Service
}

func (r *projectSlackNotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_slack_notification"
}

func (r *projectSlackNotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sends a CircleCI project's notifications to a Slack channel, as configured under " +
			"Project Settings > Slack Notifications. A project has at most one Slack channel. " +
			"Choose which notifications are sent with `circleci_project_notification_preferences`.\n\n" +
			"The organization must have the CircleCI Slack App installed. " +
			"CircleCI posts a \"Welcome! CircleCI is now configured to post updates\" message into the channel " +
			"when this resource is created, and whenever it is updated while enabled.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the channel configuration.",
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
			"channel": schema.StringAttribute{
				MarkdownDescription: "The ID of the Slack channel, such as `C0123456789`, not its name. " +
					"In Slack, it is at the bottom of the channel details.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(slackChannelIDPattern,
						"must be a Slack channel ID such as C0123456789, not a channel name"),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether notifications are posted to the channel. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"channel_name": schema.StringAttribute{
				MarkdownDescription: "The name of the Slack channel, when CircleCI reports it.",
				Computed:            true,
			},
			"disable_reason": schema.StringAttribute{
				MarkdownDescription: "Why CircleCI disabled the channel, when it did.",
				Computed:            true,
			},
		},
	}
}

func (m *projectSlackNotificationResourceModel) setFrom(c *notification.ChannelConfig) {
	m.Id = types.StringValue(c.ID)
	if c.OrgID != "" {
		m.OrganizationId = types.StringValue(c.OrgID)
	}
	if c.ProjectID != "" {
		m.ProjectId = types.StringValue(c.ProjectID)
	}
	m.Channel = types.StringValue(c.Target)
	m.Enabled = types.BoolValue(c.Enabled)
	m.ChannelName = types.StringPointerValue(c.ChannelName)
	m.DisableReason = types.StringPointerValue(c.DisableReason)
}

func (r *projectSlackNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectSlackNotificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := r.client.CreateSlackChannel(ctx,
		plan.OrganizationId.ValueString(), plan.ProjectId.ValueString(),
		plan.Channel.ValueString(), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error creating CircleCI project Slack notification",
			fmt.Sprintf("%s\n\nA project can have only one Slack channel. If it already has one, import it with "+
				"the ID %s/%s. The organization must also have the CircleCI Slack App installed.",
				err, plan.OrganizationId.ValueString(), plan.ProjectId.ValueString()))
		return
	}
	plan.setFrom(c)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *projectSlackNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectSlackNotificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := r.client.GetChannelConfig(ctx, state.Id.ValueString())
	if client.IsNotFound(err) {
		// Deleted outside Terraform: plan to create it again instead of
		// failing the refresh.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading CircleCI project Slack notification "+state.Id.ValueString(), err.Error())
		return
	}
	state.setFrom(c)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectSlackNotificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state projectSlackNotificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An update posts the Slack welcome message again whenever the config
	// stays enabled, so only send one when something the API stores changed.
	if plan.Channel.Equal(state.Channel) && plan.Enabled.Equal(state.Enabled) {
		plan.ChannelName = state.ChannelName
		plan.DisableReason = state.DisableReason
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	c, err := r.client.UpdateChannelConfig(ctx, state.Id.ValueString(), plan.Channel.ValueString(), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error updating CircleCI project Slack notification "+state.Id.ValueString(), err.Error())
		return
	}
	plan.setFrom(c)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectSlackNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectSlackNotificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteChannelConfig(ctx, state.Id.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting CircleCI project Slack notification "+state.Id.ValueString(), err.Error())
	}
}

func (r *projectSlackNotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureNotificationService(req, resp)
}

// ImportState accepts "<organization_id>/<project_id>", like the project's
// notification preferences, since a project has at most one Slack channel
// and its config ID is not shown anywhere in the UI.
func (r *projectSlackNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, ok := splitImportID(req.ID, 2)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected an import ID of the form organization_id/project_id, got %q.", req.ID))
		return
	}
	orgID, projectID := parts[0], parts[1]

	configs, err := r.client.ListProjectChannelConfigs(ctx, orgID, projectID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing CircleCI project Slack notifications", err.Error())
		return
	}
	var found *notification.ChannelConfig
	for i := range configs {
		if configs[i].ChannelType == "slack" {
			found = &configs[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Cannot import CircleCI project Slack notification",
			fmt.Sprintf("Project %s has no Slack notification channel configured.", projectID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), orgID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
}

func configureNotificationService(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *notification.Service {
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
	return wrapper.NotificationService
}
