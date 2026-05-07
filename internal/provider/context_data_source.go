// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	ccicontext "github.com/CircleCI-Public/circleci-sdk-go/context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ContextDataSource{}
	_ datasource.DataSourceWithConfigure = &ContextDataSource{}
)

// contextDataSourceModel maps the output schema.
type contextDataSourceModel struct {
	Id           types.String                 `tfsdk:"id"`
	Name         types.String                 `tfsdk:"name"`
	CreatedAt    types.String                 `tfsdk:"created_at"`
	Restrictions []restrictionDataSourceModel `tfsdk:"restrictions"`
}

type restrictionDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Value     types.String `tfsdk:"value"`
}

// NewContextDataSource is a helper function to simplify the provider implementation.
func NewContextDataSource() datasource.DataSource {
	return &ContextDataSource{}
}

// contextDataSource is the data source implementation.
type ContextDataSource struct {
	client *ccicontext.ContextService
}

// Metadata returns the data source type name.
func (d *ContextDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context"
}

// Schema defines the schema for the data source.
func (d *ContextDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a CircleCI context, including its restrictions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the context.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the context.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the context was created.",
				Computed:            true,
			},
			"restrictions": schema.ListNestedAttribute{
				MarkdownDescription: "The access restrictions for this context.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the restriction.",
							Computed:            true,
						},
						"project_id": schema.StringAttribute{
							MarkdownDescription: "The project ID associated with the restriction.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name associated with the restriction.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of restriction.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The value associated with the restriction type.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ContextDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var contextState contextDataSourceModel
	diags := req.Config.Get(ctx, &contextState)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	if contextState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing context id",
			"Missing context id",
		)
		return
	}

	context, err := d.client.Get(ctx, contextState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context with id "+contextState.Id.ValueString(),
			err.Error(),
		)
		return
	}

	restrictions, err := d.client.GetRestrictions(ctx, contextState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context restrictions",
			err.Error(),
		)
		return
	}

	// Fill restrictions
	restrictionsAttributeValues := make([]restrictionDataSourceModel, len(restrictions))
	for index, elem := range restrictions {
		restrictionsAttributeValues[index] =
			restrictionDataSourceModel{
				Id:        types.StringValue(elem.ID),
				Name:      types.StringValue(elem.Name),
				ProjectId: types.StringValue(elem.ProjectId),
				Type:      types.StringValue(elem.RestrictionType),
				Value:     types.StringValue(elem.RestrictionValue),
			}
	}

	// Map response body to model
	contextState = contextDataSourceModel{
		Id:           types.StringValue(context.ID),
		Name:         types.StringValue(context.Name),
		CreatedAt:    types.StringValue(context.CreatedAt),
		Restrictions: restrictionsAttributeValues,
	}

	// Set state
	diags = resp.State.Set(ctx, &contextState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *ContextDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *circleciClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client.ContextService
}
