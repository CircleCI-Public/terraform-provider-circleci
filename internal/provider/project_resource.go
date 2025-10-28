// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/common"
	"github.com/CircleCI-Public/circleci-sdk-go/project"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &projectResource{}
	_ resource.ResourceWithConfigure = &projectResource{}
)

// projectResourceModel maps the output schema.
type projectResourceModel struct {
	Id                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Slug                      types.String `tfsdk:"slug"`
	OrganizationName          types.String `tfsdk:"organization_name"`
	OrganizationSlug          types.String `tfsdk:"organization_slug"`
	OrganizationId            types.String `tfsdk:"organization_id"`
	VcsInfoUrl                types.String `tfsdk:"vcs_info_url"`
	VcsInfoProvider           types.String `tfsdk:"vcs_info_provider"`
	VcsInfoDefaultBranch      types.String `tfsdk:"vcs_info_default_branch"`
	AutoCancelBuilds          types.Bool   `tfsdk:"auto_cancel_builds"`
	BuildForkPrs              types.Bool   `tfsdk:"build_fork_prs"`
	DisableSSH                types.Bool   `tfsdk:"disable_ssh"`
	ForksReceiveSecretEnvVars types.Bool   `tfsdk:"forks_receive_secret_env_vars"`
	//OSS                        types.Bool   `tfsdk:"oss"`
	SetGithubStatus            types.Bool `tfsdk:"set_github_status"`
	SetupWorkflows             types.Bool `tfsdk:"setup_workflows"`
	WriteSettingsRequiresAdmin types.Bool `tfsdk:"write_settings_requires_admin"`
	PROnlyBranchOverrides      types.List `tfsdk:"pr_only_branch_overrides"`
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
				Optional:            true,
			},
			"build_fork_prs": schema.BoolAttribute{
				MarkdownDescription: "build_fork_prs configuration of the circleci project",
				Optional:            true,
			},
			"disable_ssh": schema.BoolAttribute{
				MarkdownDescription: "disable_ssh configuration of the circleci project",
				Optional:            true,
			},
			"forks_receive_secret_env_vars": schema.BoolAttribute{
				MarkdownDescription: "forks_receive_secret_env_vars configuration of the circleci project",
				Optional:            true,
			},
			/*"oss": schema.BoolAttribute{
				MarkdownDescription: "oss configuration of the circleci project",
				Optional:            true,
			},*/
			"set_github_status": schema.BoolAttribute{
				MarkdownDescription: "set_github_status configuration of the circleci project",
				Optional:            true,
			},
			"setup_workflows": schema.BoolAttribute{
				MarkdownDescription: "setup_workflows configuration of the circleci project",
				Optional:            true,
			},
			"write_settings_requires_admin": schema.BoolAttribute{
				MarkdownDescription: "write_settings_requires_admin configuration of the circleci project",
				Optional:            true,
			},
			"pr_only_branch_overrides": schema.ListAttribute{
				MarkdownDescription: "pr_only_branch_overrides configuration of the circleci project",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new context
	newCreatedProject, err := r.client.Create(
		plan.Name.ValueString(),
		plan.OrganizationId.ValueString(),
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
	if !plan.AutoCancelBuilds.IsNull() {
		newAdvancedSettings.AutocancelBuilds = plan.AutoCancelBuilds.ValueBoolPointer()
	} else {
		newAdvancedSettings.AutocancelBuilds = common.Bool(false)
	}

	if !plan.BuildForkPrs.IsNull() {
		newAdvancedSettings.BuildForkPrs = plan.BuildForkPrs.ValueBoolPointer()
	} else {
		newAdvancedSettings.BuildForkPrs = common.Bool(false)
	}

	if !plan.DisableSSH.IsNull() {
		newAdvancedSettings.DisableSSH = plan.DisableSSH.ValueBoolPointer()
	} else {
		newAdvancedSettings.DisableSSH = common.Bool(false)
	}

	/*if !plan.OSS.IsNull() {
		newAdvancedSettings.OSS = plan.OSS.ValueBoolPointer()
	} else {
		newAdvancedSettings.OSS = common.Bool(false)
	}*/

	if !plan.ForksReceiveSecretEnvVars.IsNull() {
		newAdvancedSettings.ForksReceiveSecretEnvVars = plan.ForksReceiveSecretEnvVars.ValueBoolPointer()
	} else {
		newAdvancedSettings.ForksReceiveSecretEnvVars = common.Bool(true)
	}

	if !plan.SetGithubStatus.IsNull() {
		newAdvancedSettings.SetGithubStatus = plan.SetGithubStatus.ValueBoolPointer()
	} else {
		newAdvancedSettings.SetGithubStatus = common.Bool(false)
	}

	if !plan.SetupWorkflows.IsNull() {
		newAdvancedSettings.SetupWorkflows = plan.SetupWorkflows.ValueBoolPointer()
	} else {
		newAdvancedSettings.SetupWorkflows = common.Bool(false)
	}

	if !plan.WriteSettingsRequiresAdmin.IsNull() {
		newAdvancedSettings.WriteSettingsRequiresAdmin = plan.WriteSettingsRequiresAdmin.ValueBoolPointer()
	} else {
		newAdvancedSettings.WriteSettingsRequiresAdmin = common.Bool(false)
	}

	if !plan.PROnlyBranchOverrides.IsNull() {
		prElements := plan.PROnlyBranchOverrides.Elements()
		branches := make([]string, len(prElements))
		for index, branch := range prElements {
			branches[index] = branch.String()
		}
		newAdvancedSettings.PROnlyBranchOverrides = branches
	} else {
		newAdvancedSettings.PROnlyBranchOverrides = []string{"main"}
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringValue(newCreatedProject.Id)
	plan.Name = types.StringValue(newCreatedProject.Name)
	plan.Slug = types.StringValue(newCreatedProject.Slug)
	plan.OrganizationName = types.StringValue(newCreatedProject.OrganizationName)
	plan.OrganizationSlug = types.StringValue(newCreatedProject.OrganizationSlug)
	plan.OrganizationId = types.StringValue(newCreatedProject.OrganizationId)
	plan.VcsInfoUrl = types.StringValue(newCreatedProject.VcsInfo.VcsUrl)
	plan.VcsInfoProvider = types.StringValue(newCreatedProject.VcsInfo.Provider)
	plan.VcsInfoDefaultBranch = types.StringValue(newCreatedProject.VcsInfo.DefaultBranch)

	slug := strings.Split(newCreatedProject.Slug, "/")
	newProjectSettings, err := r.client.UpdateSettings(
		project.ProjectSettings{Advanced: newAdvancedSettings},
		slug[0],
		slug[1],
		slug[2],
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating CircleCI project settings",
			fmt.Sprintf(
				`Could not update recently created CircleCI project settings:
settings: %+v
settings auto cancel builds: %+v
settings build fork: %+v
settings disable SSH: %+v
settings fork receive: %+v
settings set githubstatus: %+v
settings set workflows: %+v
settings write requires admin: %+v

org: %s
project_id: %s
project_name: %s
slug: %s
Unexpected error: %s`,
				newAdvancedSettings,
				*newAdvancedSettings.AutocancelBuilds,
				*newAdvancedSettings.BuildForkPrs,
				*newAdvancedSettings.DisableSSH,
				*newAdvancedSettings.ForksReceiveSecretEnvVars,
				*newAdvancedSettings.SetGithubStatus,
				*newAdvancedSettings.SetupWorkflows,
				*newAdvancedSettings.WriteSettingsRequiresAdmin,
				plan.OrganizationId.ValueString(),
				newCreatedProject.Id,
				newCreatedProject.Name,
				newCreatedProject.Slug,
				err.Error(),
			),
		)
		return
	}

	plan.AutoCancelBuilds = types.BoolPointerValue(newProjectSettings.Advanced.AutocancelBuilds)
	plan.BuildForkPrs = types.BoolPointerValue(newProjectSettings.Advanced.BuildForkPrs)
	plan.DisableSSH = types.BoolPointerValue(newProjectSettings.Advanced.DisableSSH)
	plan.ForksReceiveSecretEnvVars = types.BoolPointerValue(newProjectSettings.Advanced.ForksReceiveSecretEnvVars)
	//plan.OSS = types.BoolPointerValue(newProjectSettings.Advanced.OSS)
	plan.SetGithubStatus = types.BoolPointerValue(newProjectSettings.Advanced.SetGithubStatus)
	plan.SetupWorkflows = types.BoolPointerValue(newProjectSettings.Advanced.SetupWorkflows)
	plan.WriteSettingsRequiresAdmin = types.BoolPointerValue(newProjectSettings.Advanced.WriteSettingsRequiresAdmin)

	nBranchLength := len(newProjectSettings.Advanced.PROnlyBranchOverrides)
	listStringValuesBanches := make([]attr.Value, nBranchLength)
	for index, elem := range newProjectSettings.Advanced.PROnlyBranchOverrides {
		listStringValuesBanches[index] = types.StringValue(elem)
	}
	plan.PROnlyBranchOverrides, diags = types.ListValue(
		types.StringType,
		listStringValuesBanches,
	)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var projectState projectResourceModel
	diags := req.State.Get(ctx, &projectState)

	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

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

	tflog.Error(ctx, fmt.Sprintf("DAVID READ PROJECT%+v", apiProject))

	// Map response body to model
	projectState.Id = types.StringValue(apiProject.Id)
	projectState.Name = types.StringValue(apiProject.Name)

	slug := strings.Split(projectState.Slug.ValueString(), "/")
	projectSettings, err := r.client.GetSettings(
		slug[0],
		slug[1],
		slug[2],
	)

	tflog.Error(ctx, fmt.Sprintf("DAVID READ PROJECT SETTINGS %+v", projectSettings))
	tflog.Error(ctx, fmt.Sprintf("DAVID READ PROJECT SETTINGS OSS %+v", *projectSettings.Advanced.OSS))

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
	//projectState.OSS = types.BoolPointerValue(projectSettings.Advanced.OSS)
	projectState.SetGithubStatus = types.BoolPointerValue(projectSettings.Advanced.SetGithubStatus)
	projectState.SetupWorkflows = types.BoolPointerValue(projectSettings.Advanced.SetupWorkflows)
	projectState.WriteSettingsRequiresAdmin = types.BoolPointerValue(projectSettings.Advanced.WriteSettingsRequiresAdmin)

	pROnlyBranchOverridesAttributeValues := make([]attr.Value, len(projectSettings.Advanced.PROnlyBranchOverrides))
	for index, elem := range projectSettings.Advanced.PROnlyBranchOverrides {
		pROnlyBranchOverridesAttributeValues[index] = types.StringValue(elem)
	}
	PROnlyBranchOverridesListValue, _ := types.ListValue(types.StringType, pROnlyBranchOverridesAttributeValues)
	projectState.PROnlyBranchOverrides = PROnlyBranchOverridesListValue

	// Set state
	diags = resp.State.Set(ctx, &projectState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	/*var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	prOnlybranchOverrides := make([]string, len(plan.PROnlyBranchOverrides.Elements()))
	for index, elem := range plan.PROnlyBranchOverrides.Elements() {
		prOnlybranchOverrides[index] = elem.String()
	}
	advanceSettings := project.AdvanceSettings{
		AutocancelBuilds:           plan.AutoCancelBuilds.ValueBoolPointer(),
		BuildForkPrs:               plan.BuildForkPrs.ValueBoolPointer(),
		DisableSSH:                 plan.DisableSSH.ValueBoolPointer(),
		ForksReceiveSecretEnvVars:  plan.ForksReceiveSecretEnvVars.ValueBoolPointer(),
		OSS:                        plan.OSS.ValueBoolPointer(),
		SetGithubStatus:            plan.SetGithubStatus.ValueBoolPointer(),
		SetupWorkflows:             plan.SetupWorkflows.ValueBoolPointer(),
		WriteSettingsRequiresAdmin: plan.WriteSettingsRequiresAdmin.ValueBoolPointer(),
		PROnlyBranchOverrides:      prOnlybranchOverrides,
	}
	slug := strings.Split(state.Slug.ValueString(), "/")
	projectSettings := project.ProjectSettings{
		Advanced: advanceSettings,
	}
	updatedProject, err := r.client.UpdateSettings(projectSettings, slug[0], slug[1], slug[2])
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update CircleCI project settings for project: "+state.Slug.String(),
			err.Error(),
		)
		return
	}
	state.AutoCancelBuilds = types.BoolPointerValue(updatedProject.Advanced.AutocancelBuilds)
	state.BuildForkPrs = types.BoolPointerValue(updatedProject.Advanced.BuildForkPrs)
	state.DisableSSH = types.BoolPointerValue(updatedProject.Advanced.DisableSSH)
	state.ForksReceiveSecretEnvVars = types.BoolPointerValue(updatedProject.Advanced.ForksReceiveSecretEnvVars)
	state.OSS = types.BoolPointerValue(updatedProject.Advanced.OSS)
	state.SetGithubStatus = types.BoolPointerValue(updatedProject.Advanced.SetGithubStatus)
	state.SetupWorkflows = types.BoolPointerValue(updatedProject.Advanced.SetupWorkflows)
	state.WriteSettingsRequiresAdmin = types.BoolPointerValue(updatedProject.Advanced.WriteSettingsRequiresAdmin)

	pROnlyBranchOverridesAttributeValues := make([]attr.Value, len(updatedProject.Advanced.PROnlyBranchOverrides))
	for index, elem := range projectSettings.Advanced.PROnlyBranchOverrides {
		pROnlyBranchOverridesAttributeValues[index] = types.StringValue(elem)
	}
	PROnlyBranchOverridesListValue, _ := types.ListValue(types.StringType, pROnlyBranchOverridesAttributeValues)
	state.PROnlyBranchOverrides = PROnlyBranchOverridesListValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)*/
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
