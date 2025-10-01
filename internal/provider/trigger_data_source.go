// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/trigger"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &TriggerDataSource{}
	_ datasource.DataSourceWithConfigure = &TriggerDataSource{}
)

// TriggerDataSourceModel maps the output schema.
type triggerDataSourceModel struct {
	Id                              types.String `tfsdk:"id"`
	ProjectId                       types.String `tfsdk:"project_id"`
	Name                            types.String `tfsdk:"name"`
	Description                     types.String `tfsdk:"description"`
	CreatedAt                       types.String `tfsdk:"created_at"`
	CheckoutRef                     types.String `tfsdk:"checkout_ref"`
	EventName                       types.String `tfsdk:"event_name"`
	EventPreset                     types.String `tfsdk:"event_preset"`
	EventSourceProvider             types.String `tfsdk:"event_source_provider"`
	EventSourceRepositoryName       types.String `tfsdk:"event_source_repository_name"`
	EventSourceRepositoryExternalId types.String `tfsdk:"event_source_repository_external_id"`
	EventSourceWebHookUrl           types.String `tfsdk:"event_source_webhook_url"`
	Disabled                        types.Bool   `tfsdk:"disabled"`
}

// NewTriggerDataSource is a helper function to simplify the provider implementation.
func NewTriggerDataSource() datasource.DataSource {
	return &TriggerDataSource{}
}

// TriggerDataSource is the data source implementation.
type TriggerDataSource struct {
	client *trigger.TriggerService
}

// Metadata returns the data source type name.
func (d *TriggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

// Schema defines the schema for the data source.
func (d *TriggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci Trigger",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "project_id of the circleci Trigger",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci Trigger",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "description of the circleci Trigger",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci Trigger",
				Computed:            true,
			},
			"checkout_ref": schema.StringAttribute{
				MarkdownDescription: "checkout_ref of the circleci Trigger",
				Computed:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "disabled of the circleci Trigger",
				Optional:            true,
			},
			"event_name": schema.StringAttribute{
				MarkdownDescription: "event_name of the circleci trigger",
				Computed:            true,
			},
			"event_preset": schema.StringAttribute{
				MarkdownDescription: "event_preset of the circleci Trigger",
				Computed:            true,
			},
			"event_source_provider": schema.StringAttribute{
				MarkdownDescription: "event_source_provider of the circleci Trigger",
				Computed:            true,
			},
			"event_source_repository_name": schema.StringAttribute{
				MarkdownDescription: "event_source_repository_name of the circleci Trigger",
				Computed:            true,
			},
			"event_source_repository_external_id": schema.StringAttribute{
				MarkdownDescription: "event_source_repository_external_id of the circleci Trigger",
				Computed:            true,
			},
			"event_source_webhook_url": schema.StringAttribute{
				MarkdownDescription: "event_source_webhook_url of the circleci Trigger",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *TriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var triggerState triggerDataSourceModel
	diags := req.Config.Get(ctx, &triggerState)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	if triggerState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing trigger id",
			"Missing trigger id",
		)
		return
	}

	if triggerState.ProjectId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing trigger project_id",
			"Missing trigger project_id",
		)
		return
	}

	retrievedTrigger, err := d.client.Get(triggerState.ProjectId.ValueString(), triggerState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI Trigger with id "+triggerState.Id.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	triggerState = triggerDataSourceModel{
		Id:                              types.StringValue(retrievedTrigger.ID),
		ProjectId:                       triggerState.ProjectId,
		Name:                            types.StringValue(retrievedTrigger.Name),
		Description:                     types.StringValue(retrievedTrigger.Description),
		CreatedAt:                       types.StringValue(retrievedTrigger.CreatedAt),
		CheckoutRef:                     types.StringValue(retrievedTrigger.CheckoutRef),
		Disabled:                        types.BoolValue(*retrievedTrigger.Disabled),
		EventName:                       types.StringValue(retrievedTrigger.EventName),
		EventPreset:                     types.StringValue(retrievedTrigger.EventPreset),
		EventSourceProvider:             types.StringValue(retrievedTrigger.EventSource.Provider),
		EventSourceRepositoryName:       types.StringValue(retrievedTrigger.EventSource.Repo.FullName),
		EventSourceRepositoryExternalId: types.StringValue(retrievedTrigger.EventSource.Repo.ExternalId),
		EventSourceWebHookUrl:           types.StringValue(retrievedTrigger.EventSource.Webhook.Url),
	}

	// Set state
	diags = resp.State.Set(ctx, &triggerState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *TriggerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *CircleCiClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client.TriggerService
}
