// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/client"
	"terraform-provider-circleci/internal/circleci/common"
	"terraform-provider-circleci/internal/circleci/pipeline"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &pipelineResource{}
	_ resource.ResourceWithConfigure      = &pipelineResource{}
	_ resource.ResourceWithImportState    = &pipelineResource{}
	_ resource.ResourceWithValidateConfig = &pipelineResource{}
)

// pipelineResourceModel maps the output schema.
type pipelineResourceModel struct {
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

// NewPipelineResource is a helper function to simplify the provider implementation.
func NewPipelineResource() resource.Resource {
	return &pipelineResource{}
}

// pipelineResource is the resource implementation.
type pipelineResource struct {
	client *pipeline.PipelineService
}

// Metadata returns the resource type name.
func (r *pipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

// Schema defines the schema for the resource.
func (r *pipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CircleCI pipeline definition. A pipeline definition specifies where to find the pipeline configuration and where to check out code from. " +
			"CircleCI resolves a `forgejo` repository through the organization's Forgejo connection when the pipeline is created, so make the pipeline depend on the `circleci_forgejo_connection` resource; destroying that connection also deletes its Forgejo pipelines.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the pipeline.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project this pipeline belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the pipeline. Changing this value forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'name' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the pipeline.",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the pipeline was created.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_source_provider": schema.StringAttribute{
				MarkdownDescription: "The VCS provider for the pipeline's configuration source. Must be one of `github_app`, `github_server` or `forgejo`. A `forgejo` pipeline needs the organization's `circleci_forgejo_connection`, and its config and checkout sources must both be `forgejo`. Changing this value to or from `forgejo` forces a new resource.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("github_app", "github_server", pipeline.ProviderForgejo),
				},
				PlanModifiers: []planmodifier.String{forgejoProviderRequiresReplace()},
			},
			"config_source_file_path": schema.StringAttribute{
				MarkdownDescription: "The path to the pipeline configuration file within the repository.",
				Required:            true,
			},
			"config_source_repo_full_name": schema.StringAttribute{
				MarkdownDescription: "The full name of the repository containing the pipeline configuration. Can only be set for `forgejo`, where it identifies the repository when `config_source_repo_external_id` is not set; changing it forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					forgejoRepoRequiresReplace("config_source_provider"),
					repoUseStateForUnknown("config_source_repo_external_id", "config_source_provider"),
				},
			},
			"config_source_repo_external_id": schema.StringAttribute{
				MarkdownDescription: "The external ID of the repository containing the pipeline configuration. Required for `github_app` and `github_server`. For `forgejo` it is the Forgejo repository ID, and either it or `config_source_repo_full_name` must be set; changing it forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					forgejoRepoRequiresReplace("config_source_provider"),
					repoUseStateForUnknown("config_source_repo_full_name", "config_source_provider"),
				},
			},
			"checkout_source_provider": schema.StringAttribute{
				MarkdownDescription: "The VCS provider for the pipeline's checkout source. Must be one of `github_app`, `github_server` or `forgejo`. Changing this value to or from `forgejo` forces a new resource.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("github_app", "github_server", pipeline.ProviderForgejo),
				},
				PlanModifiers: []planmodifier.String{forgejoProviderRequiresReplace()},
			},
			"checkout_source_repo_full_name": schema.StringAttribute{
				MarkdownDescription: "The full name of the repository used for code checkout. Can only be set for `forgejo`, where it identifies the repository when `checkout_source_repo_external_id` is not set; changing it forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					forgejoRepoRequiresReplace("checkout_source_provider"),
					repoUseStateForUnknown("checkout_source_repo_external_id", "checkout_source_provider"),
				},
			},
			"checkout_source_repo_external_id": schema.StringAttribute{
				MarkdownDescription: "The external ID of the repository to check out code from. Required for `github_app` and `github_server`. For `forgejo` it is the Forgejo repository ID, and either it or `checkout_source_repo_full_name` must be set; changing it forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					forgejoRepoRequiresReplace("checkout_source_provider"),
					repoUseStateForUnknown("checkout_source_repo_full_name", "checkout_source_provider"),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *pipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan pipelineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configRepo := common.Repo{
		FullName:   plan.ConfigSourceRepoFullName.ValueString(),
		ExternalId: plan.ConfigSourceRepoExternalId.ValueString(),
	}
	configSource := common.ConfigSource{
		Provider: plan.ConfigSourceProvider.ValueString(),
		Repo:     configRepo,
		FilePath: plan.ConfigSourceFilePath.ValueString(),
	}
	checkoutRepo := common.Repo{
		FullName:   plan.CheckoutSourceRepoFullName.ValueString(),
		ExternalId: plan.CheckoutSourceRepoExternalId.ValueString(),
	}
	checkoutSource := common.CheckoutSource{
		Provider: plan.CheckoutSourceProvider.ValueString(),
		Repo:     checkoutRepo,
	}
	newPipeline := pipeline.Pipeline{
		ID:             plan.Id.ValueString(),
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		ConfigSource:   configSource,
		CheckoutSource: checkoutSource,
	}

	// Create new pipeline
	createdPipeline, err := r.client.Create(ctx, newPipeline, plan.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI pipeline",
			"Could not create CircleCI pipeline, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringValue(createdPipeline.ID)
	// project_id is a particular attribute from the provider
	plan.Name = types.StringValue(createdPipeline.Name)
	plan.Description = types.StringValue(createdPipeline.Description)
	plan.CreatedAt = types.StringValue(createdPipeline.CreatedAt)
	plan.ConfigSourceProvider = types.StringValue(createdPipeline.ConfigSource.Provider)
	plan.ConfigSourceFilePath = types.StringValue(createdPipeline.ConfigSource.FilePath)
	plan.ConfigSourceRepoFullName = types.StringValue(createdPipeline.ConfigSource.Repo.FullName)
	plan.ConfigSourceRepoExternalId = types.StringValue(createdPipeline.ConfigSource.Repo.ExternalId)
	plan.CheckoutSourceProvider = types.StringValue(createdPipeline.CheckoutSource.Provider)
	plan.CheckoutSourceRepoFullName = types.StringValue(createdPipeline.CheckoutSource.Repo.FullName)
	plan.CheckoutSourceRepoExternalId = types.StringValue(createdPipeline.CheckoutSource.Repo.ExternalId)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *pipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var pipelineState pipelineResourceModel
	diags := req.State.Get(ctx, &pipelineState)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	if pipelineState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing pipeline id",
			"Missing pipeline id",
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

	retrievedPipeline, err := r.client.Get(ctx, pipelineState.ProjectId.ValueString(), pipelineState.Id.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context with id "+pipelineState.Id.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	pipelineState = pipelineResourceModel{
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
	diags = resp.State.Set(ctx, &pipelineState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *pipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configSource := common.ConfigSource{
		FilePath: plan.ConfigSourceFilePath.ValueString(),
	}
	checkoutRepo := common.Repo{
		ExternalId: plan.CheckoutSourceRepoExternalId.ValueString(),
	}
	checkoutSource := common.CheckoutSource{
		Provider: plan.CheckoutSourceProvider.ValueString(),
		Repo:     checkoutRepo,
	}
	updates := pipeline.Pipeline{
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		ConfigSource:   configSource,
		CheckoutSource: checkoutSource,
	}

	updatedPipeline, err := r.client.Update(ctx, updates, plan.ProjectId.ValueString(), state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update CircleCI pipeline definition with id "+state.Id.ValueString()+" and project id "+state.ProjectId.ValueString(),
			err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(updatedPipeline.ID)
	plan.Name = types.StringValue(updatedPipeline.Name)
	plan.Description = types.StringValue(updatedPipeline.Description)
	plan.CreatedAt = types.StringValue(updatedPipeline.CreatedAt)
	plan.ConfigSourceProvider = types.StringValue(updatedPipeline.ConfigSource.Provider)
	plan.ConfigSourceFilePath = types.StringValue(updatedPipeline.ConfigSource.FilePath)
	plan.ConfigSourceRepoFullName = types.StringValue(updatedPipeline.ConfigSource.Repo.FullName)
	plan.ConfigSourceRepoExternalId = types.StringValue(updatedPipeline.ConfigSource.Repo.ExternalId)
	plan.CheckoutSourceProvider = types.StringValue(updatedPipeline.CheckoutSource.Provider)
	plan.CheckoutSourceRepoFullName = types.StringValue(updatedPipeline.CheckoutSource.Repo.FullName)
	plan.CheckoutSourceRepoExternalId = types.StringValue(updatedPipeline.CheckoutSource.Repo.ExternalId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *pipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state pipelineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.Delete(ctx, state.ProjectId.ValueString(), state.Id.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi pipeline",
			"Could not delete pipeline, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *pipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	wrapper, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *circleciClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = wrapper.PipelineService
}

func (r *pipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: "PROJECT_ID/PIPELINE_ID"
	parts := strings.SplitN(req.ID, "/", 2)

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Expected import ID format: 'project_id/pipeline_id'. Got: %s", req.ID),
		)
		return
	}

	projectId := parts[0]
	pipelineId := parts[1]

	// 1. Set the primary key 'id'
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("id"), pipelineId,
	)...)

	// 2. Set the required but unreadable 'project_id'
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("project_id"), projectId,
	)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pipelineResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config pipelineResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sources := []struct {
		name                 string
		provider             types.String
		fullName, externalID types.String
	}{
		{"config_source", config.ConfigSourceProvider, config.ConfigSourceRepoFullName, config.ConfigSourceRepoExternalId},
		{"checkout_source", config.CheckoutSourceProvider, config.CheckoutSourceRepoFullName, config.CheckoutSourceRepoExternalId},
	}
	forgejo := 0
	for _, src := range sources {
		if src.provider.IsUnknown() {
			return
		}
		if src.provider.ValueString() == pipeline.ProviderForgejo {
			forgejo++
			if src.fullName.IsNull() && src.externalID.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root(src.name+"_repo_external_id"), "Missing repository",
					fmt.Sprintf("A forgejo pipeline needs %[1]s_repo_external_id or %[1]s_repo_full_name.", src.name))
			}
			continue
		}
		if src.externalID.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root(src.name+"_repo_external_id"), "Missing repository",
				fmt.Sprintf("%s_repo_external_id is required for %s.", src.name, src.provider.ValueString()))
		}
		if !src.fullName.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root(src.name+"_repo_full_name"), "Unsupported attribute",
				fmt.Sprintf("%s_repo_full_name can only be set for forgejo; for %s it is read from CircleCI.", src.name, src.provider.ValueString()))
		}
	}
	if forgejo == 1 {
		resp.Diagnostics.AddAttributeError(path.Root("checkout_source_provider"), "Mismatched providers",
			"A forgejo pipeline must use forgejo for both config_source_provider and checkout_source_provider.")
	}
}
