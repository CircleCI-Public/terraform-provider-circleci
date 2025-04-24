package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/common"
	"github.com/CircleCI-Public/circleci-sdk-go/pipeline"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &pipelineResource{}
	_ resource.ResourceWithConfigure = &pipelineResource{}
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
	resp.TypeName = req.ProviderTypeName + "_context"
}

// Schema defines the schema for the resource.
func (r *pipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "organization_id of the circleci context",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci context",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci context",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// *** This tells Terraform to replace if 'name' changes ***
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created_at of the circleci context",
				Computed:            true,
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

	repo := common.Repo{
		FullName:   plan.ConfigSourceRepoFullName.ValueString(),
		ExternalId: plan.ConfigSourceRepoExternalId.ValueString(),
	}
	configSource := common.ConfigSource{
		Provider: plan.ConfigSourceProvider.ValueString(),
		Repo:     repo,
		FilePath: plan.ConfigSourceFilePath.ValueString(),
	}
	checkoutSource := common.CheckoutSource{
		Provider: plan.ConfigSourceProvider.ValueString(),
		Repo:     repo,
	}
	newPipeline := pipeline.Pipeline{
		ID:             plan.Id.ValueString(),
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		CreatedAt:      plan.CreatedAt.ValueString(),
		ConfigSource:   configSource,
		CheckoutSource: checkoutSource,
	}

	// Create new context
	newCciContext, err := r.client.Create(newPipeline, plan.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI context",
			"Could not create CircleCI context, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.CreatedAt = types.StringValue(newCciContext.CreatedAt)
	plan.Id = types.StringValue(newCciContext.ID)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *pipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var contextState pipelineResourceModel
	req.State.Get(ctx, &contextState)

	if contextState.Id.IsNull() {
		resp.Diagnostics.AddError(
			"Missing context id",
			"Missing context id",
		)
		return
	}

	context, err := r.client.Get(contextState.ProjectId.ValueString(), contextState.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context with id "+contextState.Id.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	contextState = pipelineResourceModel{
		Id:                           types.StringValue(context.ID),
		ProjectId:                    contextState.ProjectId,
		Name:                         types.StringValue(context.Name),
		Description:                  types.StringValue(context.Description),
		CreatedAt:                    types.StringValue(context.CreatedAt),
		ConfigSourceProvider:         types.StringValue(context.ConfigSource.Provider),
		ConfigSourceFilePath:         types.StringValue(context.ConfigSource.FilePath),
		ConfigSourceRepoFullName:     types.StringValue(context.ConfigSource.Repo.FullName),
		ConfigSourceRepoExternalId:   types.StringValue(context.ConfigSource.Repo.ExternalId),
		CheckoutSourceProvider:       types.StringValue(context.CheckoutSource.Provider),
		CheckoutSourceRepoFullName:   types.StringValue(context.CheckoutSource.Repo.FullName),
		CheckoutSourceRepoExternalId: types.StringValue(context.CheckoutSource.Repo.ExternalId),
	}

	// Set state
	diags := resp.State.Set(ctx, &contextState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *pipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	err := r.client.Delete(state.ProjectId.ValueString(), state.Id.ValueString())
	if err != nil {
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

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *circleciClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client.PipelineService
}
