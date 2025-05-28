
package provider

import (
	"context"
	"fmt"

	"github.com/CircleCI-Public/circleci-sdk-go/env"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ContextEnvironmentVariableDataSource{}
	_ datasource.DataSourceWithConfigure = &ContextEnvironmentVariableDataSource{}
)

// contextEnvironmentVariableDataSourceModel maps the output schema.
type contextEnvironmentVariableDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	CreatedAt types.String `tfsdk:"created_at"`
	ContextId types.String `tfsdk:"context_id"`
}

// NewContextEnvironmentVariableDataSource is a helper function to simplify the provider implementation.
func NewContextEnvironmentVariableDataSource() datasource.DataSource {
	return &ContextEnvironmentVariableDataSource{}
}

// ContextEnvironmentVariableDataSource is the data source implementation.
type ContextEnvironmentVariableDataSource struct {
	client *env.EnvService
}

// Metadata returns the data source type name.
func (d *ContextEnvironmentVariableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_environment_variable"
}

// Schema defines the schema for the data source.
func (d *ContextEnvironmentVariableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the circleci context environment variable",
				Required:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "updated date of the circleci context environment variable",
				Computed:            true,
			},
			"context_id": schema.StringAttribute{
				MarkdownDescription: "context id of the circleci context environment variable",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "created date of the circleci context environment variable",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ContextEnvironmentVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var contextEnvironmentVariableState contextEnvironmentVariableDataSourceModel
	req.Config.Get(ctx, &contextEnvironmentVariableState)

	if contextEnvironmentVariableState.ContextId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing environment variable context id",
			"Missing environment variable context id",
		)
		return
	}

	if contextEnvironmentVariableState.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Missing environment variable name",
			"Missing environment variable name",
		)
		return
	}

	contextEnvironmentVariables, err := d.client.List(contextEnvironmentVariableState.ContextId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context environment variable with context id "+contextEnvironmentVariableState.ContextId.ValueString(),
			err.Error(),
		)
		return
	}

	// Fill restrictions
	for _, elem := range contextEnvironmentVariables {
		if elem.Variable == contextEnvironmentVariableState.Name.ValueString() {
			contextEnvironmentVariableState = contextEnvironmentVariableDataSourceModel{
				Name:      types.StringValue(elem.Variable),
				UpdatedAt: types.StringValue(elem.UpdatedAt),
				CreatedAt: types.StringValue(elem.CreatedAt),
				ContextId: types.StringValue(elem.ContextId),
			}
			break
		}
	}

	// Set state
	diags := resp.State.Set(ctx, &contextEnvironmentVariableState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *ContextEnvironmentVariableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.EnvironmentVariableService
}
