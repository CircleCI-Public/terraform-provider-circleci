// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/organization"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &organizationDataSource{}
	_ datasource.DataSourceWithConfigure = &organizationDataSource{}
)

// organizationDataSourceModel maps the output schema.
type organizationDataSourceModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Vcs  types.String `tfsdk:"vcs_type"`
}

// NewOrganizationDataSource is a helper function to simplify the provider implementation.
func NewOrganizationDataSource() datasource.DataSource {
	return &organizationDataSource{}
}

// organizationDataSource is the data source implementation.
type organizationDataSource struct {
	client *organization.OrganizationService
}

// Metadata returns the data source type name.
func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the data source.
func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The id of the organization.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the organization.",
				Computed:            true,
			},
			"vcs_type": schema.StringAttribute{
				MarkdownDescription: "The VCS provider for the organization.",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var orgState organizationDataSourceModel
	req.Config.Get(ctx, &orgState)

	if orgState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing organization id",
			"Missing organization id",
		)
		return
	}

	org, err := d.client.Get(orgState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI organization with id "+orgState.Id.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	orgState = organizationDataSourceModel{
		Id:   types.StringValue(org.ID),
		Name: types.StringValue(org.Name),
		Vcs:  types.StringValue(org.VcsType),
	}

	// Set state
	diags := resp.State.Set(ctx, &orgState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *organizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.OrganizationService
}
