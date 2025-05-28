
package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/project"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectDataSource{}
)

// projectDataSourceModel maps the output schema.
type projectDataSourceModel struct {
	Id                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Slug                       types.String `tfsdk:"slug"`
	AutoCancelBuilds           types.Bool   `tfsdk:"auto_cancel_builds"`
	BuildForkPrs               types.Bool   `tfsdk:"build_fork_prs"`
	DisableSSH                 types.Bool   `tfsdk:"disable_ssh"`
	ForksReceiveSecretEnvVars  types.Bool   `tfsdk:"forks_receive_secret_env_vars"`
	OSS                        types.Bool   `tfsdk:"oss"`
	SetGithubStatus            types.Bool   `tfsdk:"set_github_status"`
	SetupWorkflows             types.Bool   `tfsdk:"setup_workflows"`
	WriteSettingsRequiresAdmin types.Bool   `tfsdk:"write_settings_requires_admin"`
	PROnlyBranchOverrides      types.List   `tfsdk:"pr_only_branch_overrides"`
}

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
			"auto_cancel_builds": schema.BoolAttribute{
				Computed: true,
			},
			"build_fork_prs": schema.BoolAttribute{
				Computed: true,
			},
			"disable_ssh": schema.BoolAttribute{
				Computed: true,
			},
			"forks_receive_secret_env_vars": schema.BoolAttribute{
				Computed: true,
			},
			"oss": schema.BoolAttribute{
				Computed: true,
			},
			"set_github_status": schema.BoolAttribute{
				Computed: true,
			},
			"setup_workflows": schema.BoolAttribute{
				Computed: true,
			},
			"write_settings_requires_admin": schema.BoolAttribute{
				Computed: true,
			},
			"pr_only_branch_overrides": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var projectState projectDataSourceModel
	req.Config.Get(ctx, &projectState)

	if projectState.Slug.IsNull() {
		resp.Diagnostics.AddError(
			"Missing slug",
			"Missing slug",
		)
		return
	}

	project, err := d.client.Get(projectState.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project with Slug "+projectState.Slug.ValueString(),
			err.Error(),
		)
		return
	}

	slugParts := strings.Split(project.Slug, "/")
	provider := slugParts[0]
	organization := slugParts[1]
	projectName := slugParts[2]
	projectSettings, err := d.client.GetSettings(provider, organization, projectName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project settings",
			err.Error(),
		)
		return
	}

	// Map response body to model
	projectState = projectDataSourceModel{
		Id:                         types.StringValue(project.Id),
		Name:                       types.StringValue(project.Name),
		Slug:                       types.StringValue(project.Slug),
		AutoCancelBuilds:           types.BoolValue(projectSettings.Advanced.AutocancelBuilds),
		BuildForkPrs:               types.BoolValue(projectSettings.Advanced.BuildForkPrs),
		DisableSSH:                 types.BoolValue(projectSettings.Advanced.DisableSSH),
		ForksReceiveSecretEnvVars:  types.BoolValue(projectSettings.Advanced.ForksReceiveSecretEnvVars),
		OSS:                        types.BoolValue(projectSettings.Advanced.OSS),
		SetGithubStatus:            types.BoolValue(projectSettings.Advanced.SetGithubStatus),
		SetupWorkflows:             types.BoolValue(projectSettings.Advanced.SetupWorkflows),
		WriteSettingsRequiresAdmin: types.BoolValue(projectSettings.Advanced.WriteSettingsRequiresAdmin),
	}

	pROnlyBranchOverridesAttributeValues := make([]attr.Value, len(projectSettings.Advanced.PROnlyBranchOverrides))
	for index, elem := range projectSettings.Advanced.PROnlyBranchOverrides {
		pROnlyBranchOverridesAttributeValues[index] = types.StringValue(elem)
	}
	PROnlyBranchOverridesListValue, _ := types.ListValue(types.StringType, pROnlyBranchOverridesAttributeValues)
	projectState.PROnlyBranchOverrides = PROnlyBranchOverridesListValue

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
