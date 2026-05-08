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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
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
		MarkdownDescription: "Manages a CircleCI project and its advanced settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project repository. Changing this value forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The project slug in the format `vcs-type/org-name/repo-name`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "The name of the owning organization.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the owning organization.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns this project. Changing this value forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vcs_info_url": schema.StringAttribute{
				MarkdownDescription: "The VCS URL of the project repository.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_info_provider": schema.StringAttribute{
				MarkdownDescription: "The VCS provider (e.g., `github`, `bitbucket`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_info_default_branch": schema.StringAttribute{
				MarkdownDescription: "The default branch of the project repository.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_cancel_builds": schema.BoolAttribute{
				MarkdownDescription: "Whether to automatically cancel redundant builds.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"build_fork_prs": schema.BoolAttribute{
				MarkdownDescription: "Whether to build pull requests from forked repositories.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_ssh": schema.BoolAttribute{
				MarkdownDescription: "Whether to disable SSH access to builds.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"forks_receive_secret_env_vars": schema.BoolAttribute{
				MarkdownDescription: "Whether forked pull requests can access secret environment variables.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			/*"oss": schema.BoolAttribute{
				MarkdownDescription: "Whether the project is open source.",
				Optional:            true,
			},*/
			"set_github_status": schema.BoolAttribute{
				MarkdownDescription: "Whether to set GitHub commit status on builds.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"setup_workflows": schema.BoolAttribute{
				MarkdownDescription: "Whether setup workflows are enabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"write_settings_requires_admin": schema.BoolAttribute{
				MarkdownDescription: "Whether admin permissions are required to change project settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"pr_only_branch_overrides": schema.ListAttribute{
				MarkdownDescription: "List of branches that override the PR-only build setting.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
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
		ctx,
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
	}

	if !plan.WriteSettingsRequiresAdmin.IsNull() {
		newAdvancedSettings.WriteSettingsRequiresAdmin = plan.WriteSettingsRequiresAdmin.ValueBoolPointer()
	}

	if !plan.PROnlyBranchOverrides.IsNull() {
		prElements := plan.PROnlyBranchOverrides.Elements()
		branches := make([]string, len(prElements))
		for index, branch := range prElements {
			branches[index] = branch.String()
		}
		newAdvancedSettings.PROnlyBranchOverrides = branches
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
		ctx,
		project.ProjectSettings{Advanced: newAdvancedSettings},
		slug[0],
		slug[1],
		slug[2],
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating CircleCI project settings",
			fmt.Sprintf("Could not update recently created CircleCI project settings:\n\nsettings: %+v\norg: %s\nproject_id: %s\nproject_name: %s\nslug: %s\n\nUnexpected error: %s\n", newAdvancedSettings, plan.OrganizationId.ValueString(), newCreatedProject.Id, newCreatedProject.Name, newCreatedProject.Slug, err.Error()),
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

	apiProject, err := r.client.Get(ctx, projectState.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project with Slug "+projectState.Slug.ValueString(),
			err.Error(),
		)
		return
	}

	// Map response body to model
	projectState.Id = types.StringValue(apiProject.Id)
	projectState.Name = types.StringValue(apiProject.Name)
	projectState.Slug = types.StringValue(apiProject.Slug)
	projectState.OrganizationId = types.StringValue(apiProject.OrganizationId)
	projectState.OrganizationName = types.StringValue(apiProject.OrganizationName)
	projectState.OrganizationSlug = types.StringValue(apiProject.OrganizationSlug)
	projectState.VcsInfoDefaultBranch = types.StringValue(apiProject.VcsInfo.DefaultBranch)
	projectState.VcsInfoProvider = types.StringValue(apiProject.VcsInfo.Provider)
	projectState.VcsInfoUrl = types.StringValue(apiProject.VcsInfo.VcsUrl)

	slug := strings.Split(projectState.Slug.ValueString(), "/")
	projectSettings, err := r.client.GetSettings(
		ctx,
		slug[0],
		slug[1],
		slug[2],
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
	var plan projectResourceModel
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
		AutocancelBuilds:          plan.AutoCancelBuilds.ValueBoolPointer(),
		BuildForkPrs:              plan.BuildForkPrs.ValueBoolPointer(),
		DisableSSH:                plan.DisableSSH.ValueBoolPointer(),
		ForksReceiveSecretEnvVars: plan.ForksReceiveSecretEnvVars.ValueBoolPointer(),
		//OSS:                        plan.OSS.ValueBoolPointer(),
		SetGithubStatus:            plan.SetGithubStatus.ValueBoolPointer(),
		SetupWorkflows:             plan.SetupWorkflows.ValueBoolPointer(),
		WriteSettingsRequiresAdmin: plan.WriteSettingsRequiresAdmin.ValueBoolPointer(),
		PROnlyBranchOverrides:      prOnlybranchOverrides,
	}
	slug := strings.Split(state.Slug.ValueString(), "/")
	projectSettings := project.ProjectSettings{
		Advanced: advanceSettings,
	}
	updatedProject, err := r.client.UpdateSettings(ctx, projectSettings, slug[0], slug[1], slug[2])
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
	//state.OSS = types.BoolPointerValue(updatedProject.Advanced.OSS)
	state.SetGithubStatus = types.BoolPointerValue(updatedProject.Advanced.SetGithubStatus)
	state.SetupWorkflows = types.BoolPointerValue(updatedProject.Advanced.SetupWorkflows)
	state.WriteSettingsRequiresAdmin = types.BoolPointerValue(updatedProject.Advanced.WriteSettingsRequiresAdmin)

	if len(projectSettings.Advanced.PROnlyBranchOverrides) > 0 {
		pROnlyBranchOverridesAttributeValues := make([]attr.Value, len(updatedProject.Advanced.PROnlyBranchOverrides))
		for index, elem := range projectSettings.Advanced.PROnlyBranchOverrides {
			pROnlyBranchOverridesAttributeValues[index] = types.StringValue(elem)
		}
		PROnlyBranchOverridesListValue, _ := types.ListValue(types.StringType, pROnlyBranchOverridesAttributeValues)
		state.PROnlyBranchOverrides = PROnlyBranchOverridesListValue
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
	err := r.client.Delete(ctx, state.Slug.ValueString())
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

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("slug"), req.ID,
	)...)

	if resp.Diagnostics.HasError() {
		return
	}
}
