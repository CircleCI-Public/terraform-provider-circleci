// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/CircleCI-Public/circleci-sdk-go/project"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectDataSource{}
)

type projectDataSourceModel struct {
	Id               types.String                   `tfsdk:"id"`
	Name             types.String                   `tfsdk:"name"`
	OrganizationId   types.String                   `tfsdk:"organization_id"`
	OrganizationName types.String                   `tfsdk:"organization_name"`
	OrganizationSlug types.String                   `tfsdk:"organization_slug"`
	Slug             types.String                   `tfsdk:"slug"`
	VcsInfo          *projectVcsInfoDataSourceModel `tfsdk:"vcs_info"`
}

type projectVcsInfoDataSourceModel struct {
	DefaultBranch types.String `tfsdk:"default_branch"`
	Provider      types.String `tfsdk:"provider"`
	VcsUrl        types.String `tfsdk:"vcs_url"`
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
		MarkdownDescription: "Fetches information about a CircleCI project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Project ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Project name (e.g. my-repository).",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID for the project's organization.",
				Computed:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "Name of the project's organization (e.g. my-org).",
				Computed:            true,
			},
			"organization_slug": schema.StringAttribute{
				MarkdownDescription: "Slug of the project's organization (e.g. github/my-org).",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The project's slug in the format 'vcs-type/org-name/repo-name'. For example, 'github/my-org/my-repository'.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^.+/.+/.+$`),
						"must be in the format 'vcs-type/org-name/repo-name'",
					),
				},
			},
			"vcs_info": schema.SingleNestedAttribute{
				MarkdownDescription: "Attributes relating to the project's connected version control system.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"default_branch": schema.StringAttribute{
						MarkdownDescription: "The default branch of the project's connected version control system.",
						Computed:            true,
					},
					"provider": schema.StringAttribute{
						MarkdownDescription: "The provider name of the project's connected version control system.",
						Computed:            true,
					},
					"vcs_url": schema.StringAttribute{
						MarkdownDescription: "The URL of the project's connected version control system.",
						Computed:            true,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)

		return
	}

	var data projectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := d.client.Get(ctx, data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Unable to read CircleCI project (%s), got error: %s",
				data.Slug.ValueString(),
				err,
			),
		)

		return
	}

	data.Id = types.StringValue(apiResp.Id)
	data.Name = types.StringValue(apiResp.Name)
	data.OrganizationId = types.StringValue(apiResp.OrganizationId)
	data.OrganizationName = types.StringValue(apiResp.OrganizationName)
	data.OrganizationSlug = types.StringValue(apiResp.OrganizationSlug)
	data.VcsInfo = &projectVcsInfoDataSourceModel{
		DefaultBranch: types.StringValue(apiResp.VcsInfo.DefaultBranch),
		Provider:      types.StringValue(apiResp.VcsInfo.Provider),
		VcsUrl:        types.StringValue(apiResp.VcsInfo.VcsUrl),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Configure adds the provider configured client to the data source.
func (d *ProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.ProjectService
}
