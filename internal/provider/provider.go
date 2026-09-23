// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"os"

	"github.com/Codility/terraform-provider-openmetadata/internal/client"
	"github.com/Codility/terraform-provider-openmetadata/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &OpenMetadataProvider{}

// OpenMetadataProvider implements the Terraform provider for OpenMetadata.
type OpenMetadataProvider struct {
	version string
}

type OpenMetadataProviderModel struct {
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
}

// New returns a provider.Provider constructor function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OpenMetadataProvider{version: version}
	}
}

func (p *OpenMetadataProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openmetadata"
	resp.Version = p.version
}

func (p *OpenMetadataProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing OpenMetadata resources (teams, glossaries, classifications, etc.).",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "OpenMetadata server URL (e.g., https://openmetadata.example.com). Can also be set via OPENMETADATA_HOST environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "JWT authentication token for the OpenMetadata API. Can also be set via OPENMETADATA_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *OpenMetadataProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OpenMetadataProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve host: config > env
	host := os.Getenv("OPENMETADATA_HOST")
	if !config.Host.IsNull() && !config.Host.IsUnknown() {
		host = config.Host.ValueString()
	}
	if host == "" {
		resp.Diagnostics.AddError(
			"Missing OpenMetadata Host",
			"Set the 'host' attribute in the provider block or the OPENMETADATA_HOST environment variable.",
		)
		return
	}

	// Resolve token: config > env
	token := os.Getenv("OPENMETADATA_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing OpenMetadata Token",
			"Set the 'token' attribute in the provider block or the OPENMETADATA_TOKEN environment variable.",
		)
		return
	}

	c := client.NewClient(host, token)

	// Validate connectivity
	if err := c.Ping(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Unable to connect to OpenMetadata",
			"Verify host and token are correct. Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *OpenMetadataProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewTeamResource,
		resources.NewClassificationResource,
		resources.NewTagResource,
		resources.NewGlossaryResource,
		resources.NewGlossaryTermResource,
		resources.NewPolicyResource,
		resources.NewRoleResource,
		resources.NewDatabaseServiceResource,
		resources.NewDomainResource,
	}
}

func (p *OpenMetadataProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
