// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-sdk-go/runner"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &runnerResourceClassDataSource{}
	_ datasource.DataSourceWithConfigure = &runnerResourceClassDataSource{}
)

// runnerResourceClassDataSourceModel maps the data source schema.
type runnerResourceClassDataSourceModel struct {
	OrganizationId types.String `tfsdk:"organization_id"`
	ResourceClass  types.String `tfsdk:"resource_class"`
	Id             types.String `tfsdk:"id"`
	Description    types.String `tfsdk:"description"`
}

// NewRunnerResourceClassDataSource is a helper function to simplify the provider implementation.
func NewRunnerResourceClassDataSource() datasource.DataSource {
	return &runnerResourceClassDataSource{}
}

// runnerResourceClassDataSource is the data source implementation.
type runnerResourceClassDataSource struct {
	client *runner.Service
}

// Metadata returns the data source type name.
func (d *runnerResourceClassDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_resource_class"
}

// Schema defines the schema for the data source.
func (d *runnerResourceClassDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a CircleCI runner resource class.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The organization id.",
				Required:            true,
			},
			"resource_class": schema.StringAttribute{
				MarkdownDescription: "The resource class name in `namespace/name` format (e.g. `myorg/myrunner`).",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier (UUID) of the runner resource class.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the runner resource class.",
				Computed:            true,
			},
		},
	}
}

// Read fetches the resource class from the API.
func (d *runnerResourceClassDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state runnerResourceClassDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationId := state.OrganizationId.ValueString()
	uuidOrgRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidOrgRegex.MatchString(organizationId) {
		resp.Diagnostics.AddError(
			"Invalid organization_id format",
			fmt.Sprintf("Expected UUID format, got: %s", organizationId),
		)
		return
	}

	rcName := state.ResourceClass.ValueString()
	slashIdx := strings.Index(rcName, "/")
	if slashIdx == -1 {
		resp.Diagnostics.AddError(
			"Invalid resource_class format",
			fmt.Sprintf("Expected namespace/name format, got: %s", rcName),
		)
		return
	}
	namespace := rcName[:slashIdx]

	classes, err := d.client.ListResourceClasses(ctx, namespace, organizationId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading CircleCI runner resource classes",
			"Could not list runner resource classes for namespace "+namespace+": "+err.Error(),
		)
		return
	}

	var found *runner.ResourceClass
	for i := range classes {
		if classes[i].ResourceClass == rcName {
			found = &classes[i]
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Runner resource class not found",
			fmt.Sprintf("No runner resource class with name %q was found in namespace %q.", rcName, namespace),
		)
		return
	}

	state.Id = types.StringValue(found.Id)
	state.ResourceClass = types.StringValue(found.ResourceClass)
	state.Description = types.StringValue(found.Description)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Configure adds the provider configured client to the data source.
func (d *runnerResourceClassDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.RunnerService
}
