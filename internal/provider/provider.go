// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/paragor/terraform-provider-trinogateway/internal/trinogatewayclient"
)

// Ensure TrinoGatewayProvider satisfies various provider interfaces.
var _ provider.Provider = &TrinoGatewayProvider{}

// TrinoGatewayProvider defines the provider implementation.
type TrinoGatewayProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TrinoGatewayProviderModel describes the provider data model.
type TrinoGatewayProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Login    types.String `tfsdk:"login"`
	Password types.String `tfsdk:"password"`
}

func (p *TrinoGatewayProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "trinogateway"
	resp.Version = p.version
}

func (p *TrinoGatewayProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Trino gateway endpoint",
				Required:            true,
			},
			"login": schema.StringAttribute{
				MarkdownDescription: "login",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "password",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *TrinoGatewayProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TrinoGatewayProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Endpoint.IsNull() {
		resp.Diagnostics.AddError(
			"Endpoint for trino gateway client is not specify",
			"Cant configure trino gateway client: endpoint is not specified",
		)
		return
	}

	var auth *trinogatewayclient.Auth
	if !data.Login.IsNull() {
		if data.Password.IsNull() {
			resp.Diagnostics.AddError(
				"Cant configure trino gateway client auth",
				"Cant configure trino gateway client auth: if login set, password should be set too",
			)
			return
		}
		auth = &trinogatewayclient.Auth{
			Login:    data.Login.ValueString(),
			Password: data.Password.ValueString(),
		}
	}
	// Example client configuration for data sources and resources
	client, err := trinogatewayclient.NewTrinoGatewayClient(
		data.Endpoint.ValueString(),
		auth,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cant configure trino gateway client",
			fmt.Sprintf("cant configure trino gateway client: %s", err.Error()),
		)
		return
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *TrinoGatewayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBackendResource,
	}
}

func (p *TrinoGatewayProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TrinoGatewayProvider{
			version: version,
		}
	}
}
