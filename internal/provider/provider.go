// Copyright (c) HashiCorp, Inc.
// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/CircleCI-Public/circleci-sdk-go/client"
	"github.com/CircleCI-Public/circleci-sdk-go/project"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CircleCiProvider satisfies various provider interfaces.
var _ provider.Provider = &CircleCiProvider{}
var _ provider.ProviderWithFunctions = &CircleCiProvider{}
var _ provider.ProviderWithEphemeralResources = &CircleCiProvider{}

// circleciClientWrapper wraps all the services provided by the circleci API client.
type CircleCiClientWrapper struct {
	ProjectService *project.ProjectService
}

// circleciProviderModel maps provider schema data to a Go type.
type circleciProviderModel struct {
	Host types.String `tfsdk:"host"`
	Key  types.String `tfsdk:"key"`
}

// CircleCiProvider defines the provider implementation.
type CircleCiProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func (p *CircleCiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "circleci"
	resp.Version = p.version
}

func (p *CircleCiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *CircleCiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config circleciProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if config.Key.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("key"),
			"Unknown CircleCI API Key",
			"The provider cannot create the CircleCI API client as there is an unknown configuration value for the CircleCI API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CIRCLECI_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("CIRCLECI_HOST")
	key := os.Getenv("CIRCLECI_KEY")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Key.IsNull() {
		key = config.Key.ValueString()
	}

	// If host is missing, return
	// the default value
	if host == "" {
		host = "https://circleci.com/api/v2/"
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if key == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("key"),
			"Missing CircleCI API Password",
			"The provider cannot create the CircleCI API client as there is a missing or empty value for the CircleCI API password. "+
				"Set the password value in the configuration or use the CIRCLECI_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new CircleCi client using the configuration values
	circleciClient := client.NewClient(host, key)
	projectService := project.NewProjectService(circleciClient)
	// TODO: would it be possible to verify that the client is correctly configured?

	// Make the CircleCI client available during DataSource and Resource
	// type Configure methods.
	cccw := CircleCiClientWrapper{
		ProjectService: projectService,
	}
	resp.DataSourceData = &cccw
	resp.ResourceData = &cccw
}

func (p *CircleCiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *CircleCiProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *CircleCiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
	}
}

func (p *CircleCiProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CircleCiProvider{
			version: version,
		}
	}
}
