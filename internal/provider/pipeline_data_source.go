// Copyright (c) HashiCorp, Inc.
// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/pipeline"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &PipelineDataSource{}
	_ datasource.DataSourceWithConfigure = &PipelineDataSource{}
)

// projectDataSourceModel maps the output schema.
type pipelineDataSourceModel struct {
	Id                           types.String `tfsdk:"id"`
	ProjectId                    types.String `tfsdk:"project_id"`
	Name                         types.String `tfsdk:"name"`
	Description                  types.String `tfsdk:"description"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	ConfigSourceProvider         types.String `tfsdk:"config_source_provider"`
	ConfigSourceFilePath         types.String `tfsdk:"config_source_file_path"`
	ConfigSourceRepoFullName     types.String `tfsdk:"config_source_repo_full_name"`
	ConfigSourceRepoExternalId   types.String `tfsdk:"config_source_repo_external_id"`
	CheckoutSourceProvider       types.String `tfsdk:"checkout_source_provider"`
	CheckoutSourceRepoFullName   types.String `tfsdk:"checkout_source_repo_full_name"`
	CheckoutSourceRepoExternalId types.String `tfsdk:"checkout_source_repo_external_id"`
}

// NewPipelineDataSource is a helper function to simplify the provider implementation.
func NewPipelineDataSource() datasource.DataSource {
	return &PipelineDataSource{}
}

// pipelinPipelineDataSourceeDataSource is the data source implementation.
type PipelineDataSource struct {
	client *pipeline.PipelineService
}

// Metadata returns the data source type name.
func (d *PipelineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

// Schema defines the schema for the data source.
func (d *PipelineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci pipeline",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "project_id of the circleci pipeline",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci pipeline",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "description of the circleci pipeline",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci pipeline ",
				Computed:            true,
			},
			"config_source_provider": schema.StringAttribute{
				MarkdownDescription: "config_source_provider of the circleci pipeline ",
				Computed:            true,
			},
			"config_source_file_path": schema.StringAttribute{
				MarkdownDescription: "config_source_file_path of the circleci provider",
				Computed:            true,
			},
			"config_source_repo_full_name": schema.StringAttribute{
				MarkdownDescription: "config_source_repo_full_name of the circleci provider",
				Computed:            true,
			},
			"config_source_repo_external_id": schema.StringAttribute{
				MarkdownDescription: "config_source_repo_external_id of the circleci provider",
				Computed:            true,
			},
			"checkout_source_provider": schema.StringAttribute{
				MarkdownDescription: "checkout_source_provider of the circleci provider",
				Computed:            true,
			},
			"checkout_source_repo_full_name": schema.StringAttribute{
				MarkdownDescription: "checkout_source_repo_full_name of the circleci provider",
				Computed:            true,
			},
			"checkout_source_repo_external_id": schema.StringAttribute{
				MarkdownDescription: "checkout_source_repo_external_id of the circleci provider",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *PipelineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var pipelineState pipelineDataSourceModel
	req.Config.Get(ctx, &pipelineState)

	if pipelineState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing pipeline Id",
			"Missing pipeline Id",
		)
		return
	}

	if pipelineState.ProjectId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing pipeline project_id",
			"Missing pipeline project_id",
		)
		return
	}

	retrievedPipeline, err := d.client.Get(pipelineState.ProjectId.ValueString(), pipelineState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(
				"Unable to Read CircleCI Pipeline with Project ID %s and Pipeline ID %s",
				pipelineState.ProjectId.ValueString(),
				pipelineState.Id.ValueString(),
			),
			err.Error(),
		)
		return
	}

	// Map response body to model
	pipelineState = pipelineDataSourceModel{
		Id:                           types.StringValue(retrievedPipeline.ID),
		ProjectId:                    pipelineState.ProjectId,
		Name:                         types.StringValue(retrievedPipeline.Name),
		Description:                  types.StringValue(retrievedPipeline.Description),
		CreatedAt:                    types.StringValue(retrievedPipeline.CreatedAt),
		ConfigSourceProvider:         types.StringValue(retrievedPipeline.ConfigSource.Provider),
		ConfigSourceFilePath:         types.StringValue(retrievedPipeline.ConfigSource.FilePath),
		ConfigSourceRepoFullName:     types.StringValue(retrievedPipeline.ConfigSource.Repo.FullName),
		ConfigSourceRepoExternalId:   types.StringValue(retrievedPipeline.ConfigSource.Repo.ExternalId),
		CheckoutSourceProvider:       types.StringValue(retrievedPipeline.CheckoutSource.Provider),
		CheckoutSourceRepoFullName:   types.StringValue(retrievedPipeline.CheckoutSource.Repo.FullName),
		CheckoutSourceRepoExternalId: types.StringValue(retrievedPipeline.CheckoutSource.Repo.ExternalId),
	}

	// Set state
	diags := resp.State.Set(ctx, &pipelineState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *PipelineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.PipelineService
}
