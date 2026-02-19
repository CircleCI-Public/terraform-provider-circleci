// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/envproject"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ProjectEnvironmentVariableDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectEnvironmentVariableDataSource{}
)

// projectEnvironmentVariableDataSourceModel maps the output schema.
type projectEnvironmentVariableDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	ProjectSlug types.String `tfsdk:"project_slug"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

// NewProjectEnvironmentVariableDataSource is a helper function to simplify the provider implementation.
func NewProjectEnvironmentVariableDataSource() datasource.DataSource {
	return &ProjectEnvironmentVariableDataSource{}
}

// ProjectEnvironmentVariableDataSource is the data source implementation.
type ProjectEnvironmentVariableDataSource struct {
	client *envproject.EnvService
}

// Metadata returns the data source type name.
func (d *ProjectEnvironmentVariableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variable"
}

// Schema defines the schema for the data source.
func (d *ProjectEnvironmentVariableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci project environment variable",
				Required:            true,
			},
			"project_slug": schema.StringAttribute{
				MarkdownDescription: "project slug of the circleci project environment variable (e.g. circleci/org/project)",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "value of the circleci project environment variable (masked by the API)",
				Computed:            true,
				Sensitive:           true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created date of the circleci project environment variable",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ProjectEnvironmentVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectEnvironmentVariableDataSourceModel
	diags := req.Config.Get(ctx, &state)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	envVar, err := d.client.Get(ctx, state.ProjectSlug.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project environment variable "+state.Name.ValueString(),
			err.Error(),
		)
		return
	}

	state.Name = types.StringValue(envVar.Name)
	state.Value = types.StringValue(envVar.Value)
	if !envVar.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(envVar.CreatedAt.Format("2006-01-02T15:04:05.000Z"))
	} else {
		state.CreatedAt = types.StringValue("")
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *ProjectEnvironmentVariableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client.ProjectEnvironmentVariableService
}
