// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	ccicontext "terraform-provider-circleci/internal/circleci/context"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ContextDataSource{}
	_ datasource.DataSourceWithConfigure = &ContextDataSource{}
)

// contextDataSourceModel maps the output schema.
type contextDataSourceModel struct {
	Id             types.String                 `tfsdk:"id"`
	Name           types.String                 `tfsdk:"name"`
	OrganizationId types.String                 `tfsdk:"organization_id"`
	CreatedAt      types.String                 `tfsdk:"created_at"`
	Restrictions   []restrictionDataSourceModel `tfsdk:"restrictions"`
}

type restrictionDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Value     types.String `tfsdk:"value"`
}

// NewContextDataSource is a helper function to simplify the provider implementation.
func NewContextDataSource() datasource.DataSource {
	return &ContextDataSource{}
}

// contextDataSource is the data source implementation.
type ContextDataSource struct {
	client *ccicontext.ContextService
}

// Metadata returns the data source type name.
func (d *ContextDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context"
}

// Schema defines the schema for the data source.
func (d *ContextDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches information about a CircleCI context, including its restrictions. " +
			"Look the context up either by `id`, or by `name` within the organization given by `organization_id`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the context. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the context. Exactly one of `id` or `name` must be set. " +
					"Looking a context up by `name` requires `organization_id`.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("organization_id")),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization that owns the context, used to look the context up by `name`. " +
					"Context names are unique within an organization. Cannot be set together with `id`.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("id")),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the context was created.",
				Computed:            true,
			},
			"restrictions": schema.ListNestedAttribute{
				MarkdownDescription: "The access restrictions for this context.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the restriction.",
							Computed:            true,
						},
						"project_id": schema.StringAttribute{
							MarkdownDescription: "The project ID associated with the restriction.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name associated with the restriction.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of restriction.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The value associated with the restriction type.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *ContextDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var contextState contextDataSourceModel
	diags := req.Config.Get(ctx, &contextState)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}

	var context *ccicontext.Context
	if !contextState.Id.IsNull() {
		var err error
		context, err = d.client.Get(ctx, contextState.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read CircleCI context with id "+contextState.Id.ValueString(),
				err.Error(),
			)
			return
		}
	} else {
		context = d.findByName(ctx, contextState.OrganizationId.ValueString(), contextState.Name.ValueString(), resp)
		if context == nil {
			return
		}
	}

	restrictions, err := d.client.GetRestrictions(ctx, context.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read CircleCI context restrictions",
			err.Error(),
		)
		return
	}

	// Fill restrictions
	restrictionsAttributeValues := make([]restrictionDataSourceModel, len(restrictions))
	for index, elem := range restrictions {
		restrictionsAttributeValues[index] =
			restrictionDataSourceModel{
				Id:        types.StringValue(elem.ID),
				Name:      types.StringValue(elem.Name),
				ProjectId: types.StringValue(elem.ProjectId),
				Type:      types.StringValue(elem.RestrictionType),
				Value:     types.StringValue(elem.RestrictionValue),
			}
	}

	// Map response body to model
	contextState = contextDataSourceModel{
		Id:             types.StringValue(context.ID),
		Name:           types.StringValue(context.Name),
		OrganizationId: contextState.OrganizationId,
		CreatedAt:      types.StringValue(context.CreatedAt),
		Restrictions:   restrictionsAttributeValues,
	}

	// Set state
	diags = resp.State.Set(ctx, &contextState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// findByName returns the context called name in the organization, or nil after
// adding an error to resp when it cannot be found.
func (d *ContextDataSource) findByName(ctx context.Context, organizationID, name string, resp *datasource.ReadResponse) *ccicontext.Context {
	contexts, err := d.client.ListByOrganizationID(ctx, organizationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list CircleCI contexts in organization "+organizationID,
			err.Error(),
		)
		return nil
	}

	for _, c := range contexts {
		if c.Name == name {
			return &c
		}
	}

	resp.Diagnostics.AddError(
		"CircleCI context not found",
		fmt.Sprintf("No context named %q in organization %s.", name, organizationID),
	)
	return nil
}

// Configure adds the provider configured client to the data source.
func (d *ContextDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CircleCiClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *circleciClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client.ContextService
}
