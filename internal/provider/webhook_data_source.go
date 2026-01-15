// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/webhook"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &WebhookDataSource{}
	_ datasource.DataSourceWithConfigure = &WebhookDataSource{}
)

// webhookDataSourceModel maps the data source schema.
type webhookDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	VerifyTls     types.Bool   `tfsdk:"verify_tls"`
	SigningSecret types.String `tfsdk:"signing_secret"`
	ScopeId       types.String `tfsdk:"scope_id"`
	ScopeType     types.String `tfsdk:"scope_type"`
	Events        types.List   `tfsdk:"events"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

// NewWebhookDataSource is a helper function to simplify the provider implementation.
func NewWebhookDataSource() datasource.DataSource {
	return &WebhookDataSource{}
}

// WebhookDataSource is the data source implementation.
type WebhookDataSource struct {
	client *webhook.WebhookService
}

// Metadata returns the data source type name.
func (d *WebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines the schema for the data source.
func (d *WebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about an existing CircleCI webhook.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the webhook.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the webhook.",
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to which webhook payloads will be sent.",
				Computed:            true,
			},
			"verify_tls": schema.BoolAttribute{
				MarkdownDescription: "Whether to verify TLS certificates when sending payloads.",
				Computed:            true,
			},
			"signing_secret": schema.StringAttribute{
				MarkdownDescription: "The singing secret of the webhook.",
				Computed:            true,
			},
			"scope_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the scope (project) for which the webhook is configured.",
				Computed:            true,
			},
			"scope_type": schema.StringAttribute{
				MarkdownDescription: "The type of the scope.",
				Computed:            true,
			},
			"events": schema.ListAttribute{
				MarkdownDescription: "The events that will trigger the webhook.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the webhook was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the webhook was last updated.",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *WebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config webhookDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing webhook id",
			"Missing webhook id",
		)
		return
	}

	webhookData, err := d.client.Get(ctx, config.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI webhook with id "+config.Id.ValueString(),
			err.Error(),
		)
		return
	}

	if webhookData == nil {
		resp.Diagnostics.AddError(
			"Webhook not found",
			fmt.Sprintf("Webhook with ID %s not found.", config.Id.ValueString()),
		)
		return
	}

	// Convert events to types.List
	eventsAttributeValues := make([]attr.Value, len(webhookData.Events))
	for i, event := range webhookData.Events {
		eventsAttributeValues[i] = types.StringValue(event)
	}
	eventsList, diags := types.ListValue(types.StringType, eventsAttributeValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map response to state
	state := webhookDataSourceModel{
		Id:            types.StringValue(webhookData.Id),
		Name:          types.StringValue(webhookData.Name),
		Url:           types.StringValue(webhookData.Url),
		SigningSecret: types.StringValue(webhookData.SigningSecret),
		ScopeId:       types.StringValue(webhookData.Scope.Id),
		ScopeType:     types.StringValue(webhookData.Scope.Type),
		Events:        eventsList,
		CreatedAt:     types.StringValue(webhookData.CreatedAt),
		UpdatedAt:     types.StringValue(webhookData.UpdatedAt),
	}

	if webhookData.VerifyTls != nil {
		state.VerifyTls = types.BoolValue(*webhookData.VerifyTls)
	} else {
		state.VerifyTls = types.BoolValue(true)
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *WebhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.WebhookService
}
