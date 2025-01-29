package provider

import (
	"context"
	"os"

	infisical "terraform-provider-infisical/internal/client"
	infisicalDatasource "terraform-provider-infisical/internal/provider/datasource"
	infisicalResource "terraform-provider-infisical/internal/provider/resource"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
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

	Auth *authModel `tfsdk:"auth"`
}

type authModel struct {
	Oidc      *oidcAuthModel      `tfsdk:"oidc"`
	Universal *universalAuthModel `tfsdk:"universal"`
}

type oidcAuthModel struct {
	IdentityId types.String `tfsdk:"identity_id"`
}

type universalAuthModel struct {
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
				Description: "Used to point the client to fetch secrets from your self hosted instance of Infisical. If not host is provided, https://app.infisical.com is the default host. This attribute can also be set using the `INFISICAL_HOST` environment variable",
			},
			"service_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: " (DEPRECATED, Use machine identity auth), Used to fetch/modify secrets for a given project",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "(DEPRECATED, Use the `auth` attribute), Machine identity client ID. Used to fetch/modify secrets for a given project.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "(DEPRECATED, use `auth` attribute), Machine identity client secret. Used to fetch/modify secrets for a given project",
			},
			"auth": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The configuration values for authentication",
				Attributes: map[string]schema.Attribute{
					"universal": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The configuration values for Universal Auth",
						Attributes: map[string]schema.Attribute{
							"client_id": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity client ID. Used to fetch/modify secrets for a given project. This attribute can also be set using the `INFISICAL_UNIVERSAL_AUTH_CLIENT_ID` environment variable",
							},
							"client_secret": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity client secret. Used to fetch/modify secrets for a given project. This attribute can also be set using the `INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET` environment variable",
							},
						},
					},
					"oidc": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The configuration values for OIDC Auth",
						Attributes: map[string]schema.Attribute{
							"identity_id": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity ID. Used to fetch/modify secrets for a given project. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable",
							},
						},
					},
				},
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

	host := os.Getenv(infisical.INFISICAL_HOST_NAME)

	// Service Token
	serviceToken := os.Getenv(infisical.INFISICAL_SERVICE_TOKEN_NAME)

	// Machine Identity
	clientId := os.Getenv(infisical.INFISICAL_UNIVERSAL_AUTH_CLIENT_ID_NAME)
	clientSecret := os.Getenv(infisical.INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET_NAME)
	identityId := os.Getenv(infisical.INFISICAL_MACHINE_IDENTITY_ID_NAME)

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

	var authStrategy infisical.AuthStrategyType

	if config.Auth != nil {
		if config.Auth.Oidc != nil {
			authStrategy = infisical.AuthStrategy.OIDC_MACHINE_IDENTITY
			if !config.Auth.Oidc.IdentityId.IsNull() {
				identityId = config.Auth.Oidc.IdentityId.ValueString()
			}
		}

		if config.Auth.Universal != nil {
			authStrategy = infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
			if !config.Auth.Universal.ClientId.IsNull() {
				clientId = config.Auth.Universal.ClientId.ValueString()
			}
			if !config.Auth.Universal.ClientSecret.IsNull() {
				clientSecret = config.Auth.Universal.ClientSecret.ValueString()
			}
		}
	}

	client, err := infisical.NewClient(infisical.Config{HostURL: host, AuthStrategy: authStrategy, ServiceToken: serviceToken, ClientId: clientId, ClientSecret: clientSecret, IdentityId: identityId})

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
	resp.EphemeralResourceData = client

}

// DataSources defines the data sources implemented in the provider.
func (p *infisicalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		infisicalDatasource.NewSecretDataSource,
		infisicalDatasource.NewProjectDataSource,
		infisicalDatasource.NewSecretTagDataSource,
		infisicalDatasource.NewSecretFolderDataSource,
		infisicalDatasource.NewGroupsDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *infisicalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		infisicalResource.NewSecretResource,
		infisicalResource.NewProjectResource,
		infisicalResource.NewProjectUserResource,
		infisicalResource.NewProjectIdentityResource,
		infisicalResource.NewProjectRoleResource,
		infisicalResource.NewProjectIdentitySpecificPrivilegeResource,
		infisicalResource.NewProjectGroupResource,
		infisicalResource.NewProjectSecretTagResource,
		infisicalResource.NewProjectSecretFolderResource,
		infisicalResource.NewProjectEnvironmentResource,
		infisicalResource.NewIdentityResource,
		infisicalResource.NewIdentityUniversalAuthResource,
		infisicalResource.NewIdentityUniversalAuthClientSecretResource,
		infisicalResource.NewIdentityAwsAuthResource,
		infisicalResource.NewIdentityKubernetesAuthResource,
		infisicalResource.NewIdentityGcpAuthResource,
		infisicalResource.NewIdentityAzureAuthResource,
		infisicalResource.NewIdentityOidcAuthResource,
		infisicalResource.NewIntegrationGcpSecretManagerResource,
		infisicalResource.NewIntegrationAwsParameterStoreResource,
		infisicalResource.NewIntegrationAwsSecretsManagerResource,
		infisicalResource.NewIntegrationCircleCiResource,
		infisicalResource.NewIntegrationDatabricksResource,
		infisicalResource.NewSecretApprovalPolicyResource,
		infisicalResource.NewAccessApprovalPolicyResource,
		infisicalResource.NewProjectSecretImportResource,
	}
}

// EphemeralResources defines the ephemeral resources implemented in the provider.
func (p *infisicalProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		func() ephemeral.EphemeralResource {
			return infisicalResource.NewEphemeralSecretResource()
		},
	}
}
