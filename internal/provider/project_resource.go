// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/project"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &projectResource{}
	_ resource.ResourceWithConfigure = &projectResource{}
)

// projectResourceModel maps the output schema.
type projectResourceModel struct {
	Id                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Provider                   types.String `tfsdk:"project_provider"`
	Slug                       types.String `tfsdk:"slug"`
	OrganizationName           types.String `tfsdk:"organization_name"`
	OrganizationSlug           types.String `tfsdk:"organization_slug"`
	OrganizationId             types.String `tfsdk:"organization_id"`
	VcsInfoUrl                 types.String `tfsdk:"vcs_info_url"`
	VcsInfoProvider            types.String `tfsdk:"vcs_info_provider"`
	VcsInfoDefaultBranch       types.String `tfsdk:"vcs_info_default_branch"`
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

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *project.ProjectService
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of the circleci project",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci project",
				Required:            true,
			},
			"project_provider": schema.StringAttribute{
				MarkdownDescription: "provider of the circleci project (usually `circleci`)",
				Required:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "slug of the circleci project ",
				Computed:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "organization_name of the circleci project",
				Computed:            true,
			},
			"organization_slug": schema.StringAttribute{
				MarkdownDescription: "organization_slug of the circleci project",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "organization_id of the circleci project",
				Required:            true,
			},
			"vcs_info_url": schema.StringAttribute{
				MarkdownDescription: "vcs_info_url configuration of the circleci project",
				Computed:            true,
			},
			"vcs_info_provider": schema.StringAttribute{
				MarkdownDescription: "vcs_info_provider configuration of the circleci project",
				Computed:            true,
			},
			"vcs_info_default_branch": schema.StringAttribute{
				MarkdownDescription: "vcs_info_default_branch configuration of the circleci project",
				Computed:            true,
			},
			"auto_cancel_builds": schema.BoolAttribute{
				MarkdownDescription: "auto_cancel_builds configuration of the circleci project",
				Computed:            true,
			},
			"build_fork_prs": schema.BoolAttribute{
				MarkdownDescription: "build_fork_prs configuration of the circleci project",
				Computed:            true,
			},
			"disable_ssh": schema.BoolAttribute{
				MarkdownDescription: "disable_ssh configuration of the circleci project",
				Computed:            true,
			},
			"forks_receive_secret_env_vars": schema.BoolAttribute{
				MarkdownDescription: "forks_receive_secret_env_vars configuration of the circleci project",
				Computed:            true,
			},
			"oss": schema.BoolAttribute{
				MarkdownDescription: "oss configuration of the circleci project",
				Computed:            true,
			},
			"set_github_status": schema.BoolAttribute{
				MarkdownDescription: "set_github_status configuration of the circleci project",
				Computed:            true,
			},
			"setup_workflows": schema.BoolAttribute{
				MarkdownDescription: "setup_workflows configuration of the circleci project",
				Computed:            true,
			},
			"write_settings_requires_admin": schema.BoolAttribute{
				MarkdownDescription: "write_settings_requires_admin configuration of the circleci project",
				Computed:            true,
			},
			"pr_only_branch_overrides": schema.ListAttribute{
				MarkdownDescription: "pr_only_branch_overrides configuration of the circleci project",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var circleCiTerrformProjectResource projectResourceModel
	diags := req.Plan.Get(ctx, &circleCiTerrformProjectResource)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new context
	newCreatedProject, err := r.client.Create(
		circleCiTerrformProjectResource.Name.ValueString(),
		circleCiTerrformProjectResource.OrganizationId.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI project",
			"Could not create CircleCI project, unexpected error: "+err.Error(),
		)
		return
	}

	// Create project advanced settings with the new settings when they were defined
	newAdvancedSettings := project.AdvanceSettings{}
	if !circleCiTerrformProjectResource.AutoCancelBuilds.IsNull() {
		newAdvancedSettings.AutocancelBuilds = circleCiTerrformProjectResource.AutoCancelBuilds.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.BuildForkPrs.IsNull() {
		newAdvancedSettings.BuildForkPrs = circleCiTerrformProjectResource.BuildForkPrs.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.DisableSSH.IsNull() {
		newAdvancedSettings.DisableSSH = circleCiTerrformProjectResource.DisableSSH.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.ForksReceiveSecretEnvVars.IsNull() {
		newAdvancedSettings.ForksReceiveSecretEnvVars = circleCiTerrformProjectResource.ForksReceiveSecretEnvVars.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.SetGithubStatus.IsNull() {
		newAdvancedSettings.SetGithubStatus = circleCiTerrformProjectResource.SetGithubStatus.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.SetupWorkflows.IsNull() {
		newAdvancedSettings.SetupWorkflows = circleCiTerrformProjectResource.SetupWorkflows.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.WriteSettingsRequiresAdmin.IsNull() {
		newAdvancedSettings.WriteSettingsRequiresAdmin = circleCiTerrformProjectResource.WriteSettingsRequiresAdmin.ValueBoolPointer()
	}
	if !circleCiTerrformProjectResource.PROnlyBranchOverrides.IsNull() {
		prElements := circleCiTerrformProjectResource.PROnlyBranchOverrides.Elements()
		branches := make([]string, len(prElements))
		for index, branch := range prElements {
			branches[index] = branch.String()
		}
		newAdvancedSettings.PROnlyBranchOverrides = branches
	}

	// Map response body to schema and populate Computed attribute values
	circleCiTerrformProjectResource.Id = types.StringValue(newCreatedProject.Id)
	circleCiTerrformProjectResource.Name = types.StringValue(newCreatedProject.Name)
	// provider is set in the state and is not brought by the API
	circleCiTerrformProjectResource.Slug = types.StringValue(newCreatedProject.Slug)
	circleCiTerrformProjectResource.OrganizationName = types.StringValue(newCreatedProject.OrganizationName)
	circleCiTerrformProjectResource.OrganizationSlug = types.StringValue(newCreatedProject.OrganizationSlug)
	circleCiTerrformProjectResource.OrganizationId = types.StringValue(newCreatedProject.OrganizationId)
	circleCiTerrformProjectResource.VcsInfoUrl = types.StringValue(newCreatedProject.VcsInfo.VcsUrl)
	circleCiTerrformProjectResource.VcsInfoProvider = types.StringValue(newCreatedProject.VcsInfo.Provider)
	circleCiTerrformProjectResource.VcsInfoDefaultBranch = types.StringValue(newCreatedProject.VcsInfo.DefaultBranch)

	if circleCiTerrformProjectResource.Provider.ValueString() == "circleci" {
		newProjectSettings, err := r.client.UpdateSettings(
			project.ProjectSettings{Advanced: newAdvancedSettings},
			circleCiTerrformProjectResource.Provider.ValueString(),
			circleCiTerrformProjectResource.OrganizationId.ValueString(),
			newCreatedProject.Id,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating CircleCI project settings",
				fmt.Sprintf("Could not update CircleCI project settings:\n\nsettings: %+v\nprovider: %s\norg: %s\nproject_id: %s\nproject_name: %s\n\nUnexpected error: %s\n", newAdvancedSettings, circleCiTerrformProjectResource.Provider.ValueString(), circleCiTerrformProjectResource.OrganizationId.ValueString(), newCreatedProject.Id, newCreatedProject.Name, err.Error()),
			)
			return
		}

		circleCiTerrformProjectResource.AutoCancelBuilds = types.BoolPointerValue(newProjectSettings.Advanced.AutocancelBuilds)
		circleCiTerrformProjectResource.BuildForkPrs = types.BoolPointerValue(newProjectSettings.Advanced.BuildForkPrs)
		circleCiTerrformProjectResource.DisableSSH = types.BoolPointerValue(newProjectSettings.Advanced.DisableSSH)
		circleCiTerrformProjectResource.ForksReceiveSecretEnvVars = types.BoolPointerValue(newProjectSettings.Advanced.ForksReceiveSecretEnvVars)
		circleCiTerrformProjectResource.OSS = types.BoolPointerValue(newProjectSettings.Advanced.OSS)
		circleCiTerrformProjectResource.SetGithubStatus = types.BoolPointerValue(newProjectSettings.Advanced.SetGithubStatus)
		circleCiTerrformProjectResource.SetupWorkflows = types.BoolPointerValue(newProjectSettings.Advanced.SetupWorkflows)
		circleCiTerrformProjectResource.WriteSettingsRequiresAdmin = types.BoolPointerValue(newProjectSettings.Advanced.WriteSettingsRequiresAdmin)

		nBranchLength := len(newProjectSettings.Advanced.PROnlyBranchOverrides)
		listStringValuesBanches := make([]attr.Value, nBranchLength)
		for index, elem := range newProjectSettings.Advanced.PROnlyBranchOverrides {
			listStringValuesBanches[index] = types.StringValue(elem)
		}
		circleCiTerrformProjectResource.PROnlyBranchOverrides, diags = types.ListValue(
			types.StringType,
			listStringValuesBanches,
		)
	} else {
		circleCiTerrformProjectResource.AutoCancelBuilds = types.BoolValue(false)
		circleCiTerrformProjectResource.BuildForkPrs = types.BoolValue(false)
		circleCiTerrformProjectResource.DisableSSH = types.BoolValue(false)
		circleCiTerrformProjectResource.ForksReceiveSecretEnvVars = types.BoolValue(false)
		circleCiTerrformProjectResource.OSS = types.BoolValue(false)
		circleCiTerrformProjectResource.SetGithubStatus = types.BoolValue(false)
		circleCiTerrformProjectResource.SetupWorkflows = types.BoolValue(false)
		circleCiTerrformProjectResource.WriteSettingsRequiresAdmin = types.BoolValue(false)
		nBranchLength := 0
		listStringValuesBanches := make([]attr.Value, nBranchLength)
		circleCiTerrformProjectResource.PROnlyBranchOverrides, diags = types.ListValue(
			types.StringType,
			listStringValuesBanches,
		)
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, circleCiTerrformProjectResource)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var projectState projectResourceModel
	req.State.Get(ctx, &projectState)

	if projectState.Slug.IsNull() {
		resp.Diagnostics.AddError(
			"Missing slug",
			"Missing slug",
		)
		return
	}

	apiProject, err := r.client.Get(projectState.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project with Slug "+projectState.Slug.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	projectState = projectResourceModel{
		Id:                   types.StringValue(apiProject.Id),
		Name:                 types.StringValue(apiProject.Name),
		Provider:             projectState.Provider,
		Slug:                 projectState.Slug,
		OrganizationName:     projectState.OrganizationName,
		OrganizationSlug:     projectState.OrganizationSlug,
		OrganizationId:       projectState.OrganizationId,
		VcsInfoUrl:           projectState.VcsInfoUrl,
		VcsInfoProvider:      projectState.VcsInfoProvider,
		VcsInfoDefaultBranch: projectState.VcsInfoDefaultBranch,
	}

	if apiProject.VcsInfo.Provider == "circleci" {
		projectSettings, err := r.client.GetSettings(
			projectState.Provider.ValueString(),
			projectState.OrganizationId.ValueString(),
			projectState.Id.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read CircleCI project settings",
				err.Error(),
			)
			return
		}
		projectState.AutoCancelBuilds = types.BoolPointerValue(projectSettings.Advanced.AutocancelBuilds)
		projectState.BuildForkPrs = types.BoolPointerValue(projectSettings.Advanced.BuildForkPrs)
		projectState.DisableSSH = types.BoolPointerValue(projectSettings.Advanced.DisableSSH)
		projectState.ForksReceiveSecretEnvVars = types.BoolPointerValue(projectSettings.Advanced.ForksReceiveSecretEnvVars)
		projectState.OSS = types.BoolPointerValue(projectSettings.Advanced.OSS)
		projectState.SetGithubStatus = types.BoolPointerValue(projectSettings.Advanced.SetGithubStatus)
		projectState.SetupWorkflows = types.BoolPointerValue(projectSettings.Advanced.SetupWorkflows)
		projectState.WriteSettingsRequiresAdmin = types.BoolPointerValue(projectSettings.Advanced.WriteSettingsRequiresAdmin)

		pROnlyBranchOverridesAttributeValues := make([]attr.Value, len(projectSettings.Advanced.PROnlyBranchOverrides))
		for index, elem := range projectSettings.Advanced.PROnlyBranchOverrides {
			pROnlyBranchOverridesAttributeValues[index] = types.StringValue(elem)
		}
		PROnlyBranchOverridesListValue, _ := types.ListValue(types.StringType, pROnlyBranchOverridesAttributeValues)
		projectState.PROnlyBranchOverrides = PROnlyBranchOverridesListValue
	} else {
		projectState.AutoCancelBuilds = types.BoolValue(false)
		projectState.BuildForkPrs = types.BoolValue(false)
		projectState.DisableSSH = types.BoolValue(false)
		projectState.ForksReceiveSecretEnvVars = types.BoolValue(false)
		projectState.OSS = types.BoolValue(false)
		projectState.SetGithubStatus = types.BoolValue(false)
		projectState.SetupWorkflows = types.BoolValue(false)
		projectState.WriteSettingsRequiresAdmin = types.BoolValue(false)
		pROnlyBranchOverridesAttributeValues := make([]attr.Value, 0)
		PROnlyBranchOverridesListValue, _ := types.ListValue(types.StringType, pROnlyBranchOverridesAttributeValues)
		projectState.PROnlyBranchOverrides = PROnlyBranchOverridesListValue
	}

	// Set state
	diags := resp.State.Set(ctx, &projectState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing project
	err := r.client.Delete(state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting CircleCi Project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client.ProjectService
}
