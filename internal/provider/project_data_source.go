// Copyright (c) HashiCorp, Inc.
// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/project"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectDataSource{}
)

// projectDataSourceModel maps the data source schema data.
type projectDataSourceModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Slug types.String `tfsdk:"slug"`
}

// projectSettingsModel maps project settings data.
/*
type projectSettingsModel struct {
	AutoCancelBuilds           types.Bool     `tfsdk:"auto_cancel_builds"`
	BuildForkPrs               types.Bool     `tfsdk:"build_for_prs"`
	DisableSSH                 types.Bool     `tfsdk:"disable_ssh"`
	ForksReceiveSecretEnvVars  types.Bool     `tfsdk:"forks_receive_secret_env_vars"`
	OSS                        types.Bool     `tfsdk:"oss"`
	SetGithubStatus            types.Bool     `tfsdk:"set_github_status"`
	SetupWorkflows             types.Bool     `tfsdk:"setup_workflows"`
	WriteSettingsRequiresAdmin types.Bool     `tfsdk:"write_settings_requires_admin"`
	PROnlyBranchOverrides      types.ListType `tfsdk:"pr_only_branch_overrides"`
}
*/

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

// ProjectDataSource is the data source implementation.
type ProjectDataSource struct {
	client *project.ProjectService
}

// Metadata returns the data source type name.
func (d *ProjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the data source.
func (d *ProjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci project",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci project",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "slug of the circleci project",
				Required:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var projectState projectDataSourceModel
	req.Config.Get(ctx, &projectState)

	project, err := d.client.Get(projectState.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project",
			err.Error(),
		)
		return
	}

	/*
		projectSettings, err := d.client.GetSettings(provider, organization, projectName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read CircleCI project settings",
				err.Error(),
			)
			return
		}
	*/

	// Map response body to model
	projectState = projectDataSourceModel{
		Id:   types.StringValue(project.ID),
		Name: types.StringValue(project.Name),
		Slug: types.StringValue(project.Slug),
	}

	//projectState.Settings = projectSettingsModel{
	//	AutoCancelBuilds:           types.BoolValue(projectSettings.Advanced.AutocancelBuilds),
	//	BuildForkPrs:               types.BoolValue(projectSettings.Advanced.BuildForkPrs),
	//	DisableSSH:                 types.BoolValue(projectSettings.Advanced.DisableSSH),
	//	ForksReceiveSecretEnvVars:  types.BoolValue(projectSettings.Advanced.ForksReceiveSecretEnvVars),
	//	OSS:                        types.BoolValue(projectSettings.Advanced.OSS),
	//	SetGithubStatus:            types.BoolValue(projectSettings.Advanced.SetGithubStatus),
	//	SetupWorkflows:             types.BoolValue(projectSettings.Advanced.SetupWorkflows),
	//	WriteSettingsRequiresAdmin: types.BoolValue(projectSettings.Advanced.WriteSettingsRequiresAdmin),
	//	PROnlyBranchOverrides:      types.List(projectSettings.Advanced.PROnlyBranchOverrides),
	//}

	// Set state
	diags := resp.State.Set(ctx, &projectState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *ProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.ProjectService
}
