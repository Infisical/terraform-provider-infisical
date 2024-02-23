package provider

import (
	"context"
	"os"

	infisical "terraform-provider-infisical/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &infisicalProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &infisicalProvider{
			version: version,
		}
	}
}

// infisicalProvider is the provider implementation.
type infisicalProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// infisicalProviderModel maps provider schema data to a Go type.
type infisicalProviderModel struct {
	Host         types.String `tfsdk:"host"`
	ServiceToken types.String `tfsdk:"service_token"`

	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// Metadata returns the provider type name.
func (p *infisicalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "infisical"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *infisicalProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This provider allows you to interact with Infisical",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "Used to point the client to fetch secrets from your self hosted instance of Infisical. If not host is provided, https://app.infisical.com is the default host.",
			},
			"service_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: " (DEPRECATED, USE MACHINE IDENTITY), Used to fetch/modify secrets for a given project",
			},

			"client_id": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Machine identity client ID. Used to fetch/modify secrets for a given project",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Machine identity client secret. Used to fetch/modify secrets for a given project",
			},
		},
	}
}

func (p *infisicalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration

	var config infisicalProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if config.ServiceToken.IsUnknown() {
		resp.Diagnostics.AddError("No authentication credentials provided", "You must define service_token field of the provider")
	}

	host := os.Getenv("INFISICAL_HOST")

	// Service Token
	serviceToken := os.Getenv("INFISICAL_SERVICE_TOKEN")

	// Machine Identity
	clientId := os.Getenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_ID")
	clientSecret := os.Getenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.ServiceToken.IsNull() {
		serviceToken = config.ServiceToken.ValueString()
	}

	if !config.ClientId.IsNull() {
		clientId = config.ClientId.ValueString()
	}

	if !config.ClientSecret.IsNull() {
		clientSecret = config.ClientSecret.ValueString()
	}

	// set default to cloud infisical if host is empty
	if host == "" {
		host = "https://app.infisical.com"
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := infisical.NewClient(infisical.Config{HostURL: host, ServiceToken: serviceToken, ClientId: clientId, ClientSecret: clientSecret})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Infisical API Client",
			"An unexpected error occurred when creating the Infisical API client. "+
				"If the error is not clear, please get in touch at infisical.com/slack.\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

}

// DataSources defines the data sources implemented in the provider.
func (p *infisicalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSecretDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *infisicalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSecretResource,
		NewProjectResource,
	}
}
