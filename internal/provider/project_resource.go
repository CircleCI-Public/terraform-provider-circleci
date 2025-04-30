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
	OrganizationSlugPart       types.String `tfsdk:"organization_slug_part"`
	SlugPart                   types.String `tfsdk:"slug_part"`
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

func (project projectResourceModel) Slug() string {
	return fmt.Sprintf("%s/%s/%s", project.Provider, project.OrganizationSlugPart, project.SlugPart)
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
			"organization_slug_part": schema.StringAttribute{
				MarkdownDescription: "organization_slug_part of the circleci project " +
					"(an organization has a slug of the form `{provider}/{organization_slug_part}` " +
					"that this the second part of the organization's slug)",
				Required: true,
			},
			"slug_part": schema.StringAttribute{
				MarkdownDescription: "slug_part of the circleci project " +
					"(an project has a slug of the form `{provider}/{organization_slug_part}/{slug_part}` " +
					"that this the third part of the project's slug)",
				Computed: true,
			},
			"auto_cancel_builds": schema.BoolAttribute{
				MarkdownDescription: "auto_cancel_builds configurtion of the circleci provider",
				Computed:            true,
			},
			"build_fork_prs": schema.BoolAttribute{
				MarkdownDescription: "build_fork_prs configurtion of the circleci provider",
				Computed:            true,
			},
			"disable_ssh": schema.BoolAttribute{
				MarkdownDescription: "disable_ssh configurtion of the circleci provider",
				Computed:            true,
			},
			"forks_receive_secret_env_vars": schema.BoolAttribute{
				MarkdownDescription: "forks_receive_secret_env_vars configurtion of the circleci provider",
				Computed:            true,
			},
			"oss": schema.BoolAttribute{
				MarkdownDescription: "oss configurtion of the circleci provider",
				Computed:            true,
			},
			"set_github_status": schema.BoolAttribute{
				MarkdownDescription: "set_github_status configurtion of the circleci provider",
				Computed:            true,
			},
			"setup_workflows": schema.BoolAttribute{
				MarkdownDescription: "setup_workflows configurtion of the circleci provider",
				Computed:            true,
			},
			"write_settings_requires_admin": schema.BoolAttribute{
				MarkdownDescription: "write_settings_requires_admin configurtion of the circleci provider",
				Computed:            true,
			},
			"pr_only_branch_overrides": schema.ListAttribute{
				MarkdownDescription: "pr_only_branch_overrides configurtion of the circleci provider",
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
	// TODO: it is not clear if the first argument is the name of the project or the slug part, should be the name
	newProjectSettings, err := r.client.Create(
		circleCiTerrformProjectResource.Name.ValueString(),
		circleCiTerrformProjectResource.OrganizationSlugPart.ValueString(),
		circleCiTerrformProjectResource.Provider.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating CircleCI project",
			"Could not create CircleCI project, unexpected error: "+err.Error(),
		)
		return
	}

	// Update the project settings with the new settings when they were defined
	if !circleCiTerrformProjectResource.AutoCancelBuilds.IsNull() {
		newProjectSettings.Advanced.AutocancelBuilds = circleCiTerrformProjectResource.AutoCancelBuilds.ValueBool()
	}
	if !circleCiTerrformProjectResource.BuildForkPrs.IsNull() {
		newProjectSettings.Advanced.BuildForkPrs = circleCiTerrformProjectResource.BuildForkPrs.ValueBool()
	}
	if !circleCiTerrformProjectResource.DisableSSH.IsNull() {
		newProjectSettings.Advanced.DisableSSH = circleCiTerrformProjectResource.DisableSSH.ValueBool()
	}
	if !circleCiTerrformProjectResource.ForksReceiveSecretEnvVars.IsNull() {
		newProjectSettings.Advanced.ForksReceiveSecretEnvVars = circleCiTerrformProjectResource.ForksReceiveSecretEnvVars.ValueBool()
	}
	if !circleCiTerrformProjectResource.OSS.IsNull() {
		newProjectSettings.Advanced.OSS = circleCiTerrformProjectResource.OSS.ValueBool()
	}
	if !circleCiTerrformProjectResource.SetGithubStatus.IsNull() {
		newProjectSettings.Advanced.SetGithubStatus = circleCiTerrformProjectResource.SetGithubStatus.ValueBool()
	}
	if !circleCiTerrformProjectResource.SetupWorkflows.IsNull() {
		newProjectSettings.Advanced.SetupWorkflows = circleCiTerrformProjectResource.SetupWorkflows.ValueBool()
	}
	if !circleCiTerrformProjectResource.WriteSettingsRequiresAdmin.IsNull() {
		newProjectSettings.Advanced.WriteSettingsRequiresAdmin = circleCiTerrformProjectResource.WriteSettingsRequiresAdmin.ValueBool()
	}
	if !circleCiTerrformProjectResource.PROnlyBranchOverrides.IsNull() {
		prElements := circleCiTerrformProjectResource.PROnlyBranchOverrides.Elements()
		branches := make([]string, len(prElements))
		for index, branch := range prElements {
			branches[index] = branch.String()
		}
		newProjectSettings.Advanced.PROnlyBranchOverrides = branches
	}

	// TODO: it is not clear if the second argument is the name of the project or the slug part, can be any
	_, err = r.client.UpdateSettings(
		*newProjectSettings,
		circleCiTerrformProjectResource.Name.ValueString(),
		circleCiTerrformProjectResource.OrganizationSlugPart.ValueString(),
		circleCiTerrformProjectResource.Provider.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating CircleCI project settings",
			"Could not create CircleCI project, unexpected error: "+err.Error(),
		)
		return
	}

	// Get project (to get its ID given that the create method does not return it)
	// TODO: this will fail given that at this point we do not have the project's slug third section: `{provider}/{organization}/{project_slug_part}`
	newProject, err := r.client.Get(circleCiTerrformProjectResource.Slug())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting CircleCI project by slug while creating a project",
			"Could not create CircleCI project, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	circleCiTerrformProjectResource.Id = types.StringValue(newProject.ID)
	// name, provider and organization do not change outside of Terraform
	circleCiTerrformProjectResource.AutoCancelBuilds = types.BoolValue(newProjectSettings.Advanced.AutocancelBuilds)
	circleCiTerrformProjectResource.BuildForkPrs = types.BoolValue(newProjectSettings.Advanced.BuildForkPrs)
	circleCiTerrformProjectResource.DisableSSH = types.BoolValue(newProjectSettings.Advanced.DisableSSH)
	circleCiTerrformProjectResource.ForksReceiveSecretEnvVars = types.BoolValue(newProjectSettings.Advanced.ForksReceiveSecretEnvVars)
	circleCiTerrformProjectResource.OSS = types.BoolValue(newProjectSettings.Advanced.OSS)
	circleCiTerrformProjectResource.SetGithubStatus = types.BoolValue(newProjectSettings.Advanced.SetGithubStatus)
	circleCiTerrformProjectResource.SetupWorkflows = types.BoolValue(newProjectSettings.Advanced.SetupWorkflows)
	circleCiTerrformProjectResource.WriteSettingsRequiresAdmin = types.BoolValue(newProjectSettings.Advanced.WriteSettingsRequiresAdmin)

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

	if projectState.Provider.IsNull() {
		resp.Diagnostics.AddError(
			"Missing provider",
			"Missing provider",
		)
		return
	}

	if projectState.OrganizationSlugPart.IsNull() {
		resp.Diagnostics.AddError(
			"Missing organization_slug_part",
			"Missing organization_slug_part",
		)
		return
	}

	if projectState.SlugPart.IsNull() {
		resp.Diagnostics.AddError(
			"Missing project slug_part",
			"Missing project slug_part",
		)
		return
	}

	apiProject, err := r.client.Get(projectState.Slug())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project with Slug "+projectState.Slug(),
			err.Error(),
		)
		return
	}

	projectSettings, err := r.client.GetSettings(
		projectState.Provider.ValueString(),
		projectState.OrganizationSlugPart.ValueString(),
		projectState.SlugPart.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI project settings",
			err.Error(),
		)
		return
	}

	// Map response body to model
	projectState = projectResourceModel{
		Id:                         types.StringValue(apiProject.ID),
		Name:                       types.StringValue(apiProject.Name),
		Provider:                   projectState.Provider,
		OrganizationSlugPart:       projectState.OrganizationSlugPart,
		SlugPart:                   projectState.SlugPart,
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

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: Wait for sdk client to implement deletio of a project
	resp.Diagnostics.AddError(
		"Error Deleting CircleCi Project",
		"Deletion of a project is not implemented yet",
	)
	// return
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
