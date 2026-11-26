// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-circleci/internal/circleci/project"
)

var (
	_ datasource.DataSource              = &CheckoutKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &CheckoutKeysDataSource{}
)

type checkoutKeysDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	ProjectSlug types.String `tfsdk:"project_slug"`
	Keys        types.List   `tfsdk:"keys"`
}

type checkoutKeyDataSourceModel struct {
	PublicKey   types.String `tfsdk:"public_key"`
	Type        types.String `tfsdk:"type"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	Preferred   types.Bool   `tfsdk:"preferred"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewCheckoutKeysDataSource() datasource.DataSource {
	return &CheckoutKeysDataSource{}
}

type CheckoutKeysDataSource struct {
	client *project.ProjectService
}

func (d *CheckoutKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_checkout_keys"
}

func (d *CheckoutKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches checkout keys for a CircleCI project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The project slug used as the data source ID.",
				Computed:            true,
			},
			"project_slug": schema.StringAttribute{
				MarkdownDescription: "The project's slug in the format `vcs-type/org-name/repo-name`.",
				Required:            true,
			},
			"keys": schema.ListNestedAttribute{
				MarkdownDescription: "The project's checkout keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"public_key": schema.StringAttribute{
							MarkdownDescription: "The public checkout key.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The checkout key type.",
							Computed:            true,
						},
						"fingerprint": schema.StringAttribute{
							MarkdownDescription: "The checkout key fingerprint.",
							Computed:            true,
						},
						"preferred": schema.BoolAttribute{
							MarkdownDescription: "Whether the checkout key is preferred.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The timestamp when the checkout key was created.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *CheckoutKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured HTTP Client",
			"Expected configured HTTP client. Please report this issue to the provider developers.",
		)
		return
	}

	var data checkoutKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := d.client.GetCheckoutKeys(ctx, data.ProjectSlug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to read CircleCI checkout keys (%s), got error: %s", data.ProjectSlug.ValueString(), err),
		)
		return
	}

	keyValues := make([]checkoutKeyDataSourceModel, len(apiResp))
	for i, key := range apiResp {
		keyValues[i] = checkoutKeyDataSourceModel{
			PublicKey:   types.StringValue(key.PublicKey),
			Type:        types.StringValue(key.Type),
			Fingerprint: types.StringValue(key.Fingerprint),
			Preferred:   types.BoolValue(key.Preferred),
			CreatedAt:   types.StringValue(key.CreatedAt),
		}
	}

	keys, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"public_key":  types.StringType,
			"type":        types.StringType,
			"fingerprint": types.StringType,
			"preferred":   types.BoolType,
			"created_at":  types.StringType,
		},
	}, keyValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = types.StringValue(data.ProjectSlug.ValueString())
	data.Keys = keys
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *CheckoutKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client.ProjectService
}
