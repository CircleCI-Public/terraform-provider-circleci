// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/project"
)

var (
	_ datasource.DataSource              = &ProjectSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectSettingsDataSource{}
)

type projectSettingsDataSourceModel struct {
	AutoCancelBuilds           types.Bool   `tfsdk:"auto_cancel_builds"`
	BuildForkPrs               types.Bool   `tfsdk:"build_fork_prs"`
	DisableSSH                 types.Bool   `tfsdk:"disable_ssh"`
	ForksReceiveSecretEnvVars  types.Bool   `tfsdk:"forks_receive_secret_env_vars"`
	OSS                        types.Bool   `tfsdk:"oss"`
	PROnlyBranchOverrides      types.Set    `tfsdk:"pr_only_branch_overrides"`
	Slug                       types.String `tfsdk:"slug"`
	SetGithubStatus            types.Bool   `tfsdk:"set_github_status"`
	SetupWorkflows             types.Bool   `tfsdk:"setup_workflows"`
	WriteSettingsRequiresAdmin types.Bool   `tfsdk:"write_settings_requires_admin"`
}

// NewProjectSettingsDataSource is a helper function to simplify the provider implementation.
func NewProjectSettingsDataSource() datasource.DataSource {
	return &ProjectSettingsDataSource{}
}

type ProjectSettingsDataSource struct {
	client *project.ProjectService
}

func (d *ProjectSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_settings"
}

func (d *ProjectSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a CircleCI project's settings.",
		Attributes: map[string]schema.Attribute{
			"auto_cancel_builds": schema.BoolAttribute{
				MarkdownDescription: "Except for on your default branch, we will automatically cancel any outstanding workflows on a branch when a newer pipeline is triggered on that branch. Scheduled workflows and re-runs are not auto-canceled.",
				Computed:            true,
			},
			"build_fork_prs": schema.BoolAttribute{
				MarkdownDescription: "Run builds for pull requests from forks. CircleCI will automatically update the commit status shown on GitHub's pull request page.",
				Computed:            true,
			},
			"disable_ssh": schema.BoolAttribute{
				MarkdownDescription: "This will disable SSH reruns for this project.",
				Computed:            true,
			},
			"forks_receive_secret_env_vars": schema.BoolAttribute{
				MarkdownDescription: "Run builds for forked pull requests with this project's configuration, environment variables, and secrets. The build cache is also shared between the original repository and all forks.",
				Computed:            true,
			},
			"oss": schema.BoolAttribute{
				MarkdownDescription: "Organizations on our free plan get an amount of free credits per month to use for Linux open source builds. Enabling this will allow this project’s builds to use them and let others see your builds, both through the web UI and the API.",
				Computed:            true,
			},
			"pr_only_branch_overrides": schema.SetAttribute{
				MarkdownDescription: "Set of branches that override the PR-only build setting.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The project's slug in the format `vcs-type/org-name/repo-name`. For example, `github/CircleCI-Public/terraform-provider-circleci`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^.+/.+/.+$`),
						"must be in the format 'vcs-type/org-name/repo-name'",
					),
				},
			},
			"set_github_status": schema.BoolAttribute{
				MarkdownDescription: "Report the status of every pushed commit to GitHub’s status API. Updates reported per job.",
				Computed:            true,
			},
			"setup_workflows": schema.BoolAttribute{
				MarkdownDescription: "This will allow you to conditionally trigger configurations outside the primary .circleci parent directory, update pipeline parameters before a build is run, and generate your own customized configurations if defined in your config.yml.",
				Computed:            true,
			},
			"write_settings_requires_admin": schema.BoolAttribute{
				MarkdownDescription: "Whether admin permissions are required to change project settings.",
				Computed:            true,
			},
		},
	}
}

func (d *ProjectSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)

		return
	}

	var data projectSettingsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	splitSlug := strings.SplitN(data.Slug.ValueString(), "/", 3)

	apiResp, err := d.client.GetSettings(ctx, splitSlug[0], splitSlug[1], splitSlug[2])
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Unable to read CircleCI project settings (%s), got error: %s",
				data.Slug.ValueString(),
				err,
			),
		)
		return
	}

	data.AutoCancelBuilds = types.BoolPointerValue(apiResp.Advanced.AutocancelBuilds)
	data.BuildForkPrs = types.BoolPointerValue(apiResp.Advanced.BuildForkPrs)
	data.DisableSSH = types.BoolPointerValue(apiResp.Advanced.DisableSSH)
	data.ForksReceiveSecretEnvVars = types.BoolPointerValue(apiResp.Advanced.ForksReceiveSecretEnvVars)
	data.OSS = types.BoolPointerValue(apiResp.Advanced.OSS)
	data.SetGithubStatus = types.BoolPointerValue(apiResp.Advanced.SetGithubStatus)
	data.SetupWorkflows = types.BoolPointerValue(apiResp.Advanced.SetupWorkflows)
	data.WriteSettingsRequiresAdmin = types.BoolPointerValue(apiResp.Advanced.WriteSettingsRequiresAdmin)

	overrides, diags := types.SetValueFrom(ctx, types.StringType, apiResp.Advanced.PROnlyBranchOverrides)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	data.PROnlyBranchOverrides = overrides

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ProjectSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
