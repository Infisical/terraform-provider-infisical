package provider

import (
	"context"
	"os"

	infisical "terraform-provider-infisical/internal/client"
	infisicalDatasource "terraform-provider-infisical/internal/provider/datasource"
	infisicalResource "terraform-provider-infisical/internal/provider/resource"
	appConnectionResource "terraform-provider-infisical/internal/provider/resource/app_connection"
	dynamicSecretResource "terraform-provider-infisical/internal/provider/resource/dynamic_secret"
	externalKmsResource "terraform-provider-infisical/internal/provider/resource/external_kms"
	secretRotationResource "terraform-provider-infisical/internal/provider/resource/secret_rotation"
	secretSyncResource "terraform-provider-infisical/internal/provider/resource/secret_sync"

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
	Oidc       *oidcAuthModel       `tfsdk:"oidc"`
	Token      types.String         `tfsdk:"token"`
	Universal  *universalAuthModel  `tfsdk:"universal"`
	Kubernetes *kubernetesAuthModel `tfsdk:"kubernetes"`
	AWS        *awsIamAuthModel     `tfsdk:"aws_iam"`
}

type oidcAuthModel struct {
	IdentityId   types.String `tfsdk:"identity_id"`
	TokenEnvName types.String `tfsdk:"token_environment_variable_name"`
}

type universalAuthModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

type awsIamAuthModel struct {
	IdentityId types.String `tfsdk:"identity_id"`
}

type kubernetesAuthModel struct {
	IdentityId types.String `tfsdk:"identity_id"`
	TokenPath  types.String `tfsdk:"service_account_token_path"`
	Token      types.String `tfsdk:"service_account_token"`
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
					"token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "The authentication token for Machine Identity Token Auth. This attribute can also be set using the `INFISICAL_TOKEN` environment variable",
					},
					"universal": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The configuration values for Universal Auth",
						Attributes: map[string]schema.Attribute{
							"client_id": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity client ID. This attribute can also be set using the `INFISICAL_UNIVERSAL_AUTH_CLIENT_ID` environment variable",
							},
							"client_secret": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity client secret. This attribute can also be set using the `INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET` environment variable",
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
								Description: "Machine identity ID. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable",
							},
							"token_environment_variable_name": schema.StringAttribute{
								Optional:    true,
								Sensitive:   false,
								Description: "The environment variable name for the OIDC JWT token. This attribute can also be set using the `INFISICAL_OIDC_AUTH_TOKEN_KEY_NAME` environment variable. Default is `INFISICAL_AUTH_JWT`.",
							},
						},
					},
					"kubernetes": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The configuration values for Kubernetes Auth",
						Attributes: map[string]schema.Attribute{
							"identity_id": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity ID. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable",
							},
							"service_account_token_path": schema.StringAttribute{
								Optional:    true,
								Sensitive:   false,
								Description: "The path to the service account token. This attribute can also be set using the `INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH` environment variable. Default is `/var/run/secrets/kubernetes.io/serviceaccount/token`.",
							},
							"service_account_token": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "The service account token. This attribute can also be set using the `INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN` environment variable",
							},
						},
					},
					"aws_iam": schema.SingleNestedAttribute{
						Optional:    true,
						Description: "The configuration values for AWS IAM Auth",
						Attributes: map[string]schema.Attribute{
							"identity_id": schema.StringAttribute{
								Optional:    true,
								Sensitive:   true,
								Description: "Machine identity ID. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable",
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
	oidcTokenEnvName := os.Getenv(infisical.INFISICAL_OIDC_AUTH_TOKEN_NAME)
	token := os.Getenv(infisical.INFISICAL_TOKEN_NAME)
	serviceAccountToken := os.Getenv(infisical.INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN_NAME)
	serviceAccountTokenPath := os.Getenv(infisical.INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH_NAME)

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

	var authStrategy infisical.AuthStrategyType = ""

	if config.Auth != nil {
		if config.Auth.Oidc != nil {
			authStrategy = infisical.AuthStrategy.OIDC_MACHINE_IDENTITY
			if !config.Auth.Oidc.IdentityId.IsNull() {
				identityId = config.Auth.Oidc.IdentityId.ValueString()
			}

			if !config.Auth.Oidc.TokenEnvName.IsNull() {
				oidcTokenEnvName = config.Auth.Oidc.TokenEnvName.ValueString()
			}
		} else if config.Auth.Universal != nil {
			authStrategy = infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
			if !config.Auth.Universal.ClientId.IsNull() {
				clientId = config.Auth.Universal.ClientId.ValueString()
			}
			if !config.Auth.Universal.ClientSecret.IsNull() {
				clientSecret = config.Auth.Universal.ClientSecret.ValueString()
			}
		} else if config.Auth.Kubernetes != nil {
			authStrategy = infisical.AuthStrategy.KUBERNETES_MACHINE_IDENTITY
			if !config.Auth.Kubernetes.IdentityId.IsNull() {
				identityId = config.Auth.Kubernetes.IdentityId.ValueString()
			}

			if !config.Auth.Kubernetes.TokenPath.IsNull() {
				serviceAccountTokenPath = config.Auth.Kubernetes.TokenPath.ValueString()
			}

			if !config.Auth.Kubernetes.Token.IsNull() {
				serviceAccountToken = config.Auth.Kubernetes.Token.ValueString()
			}
		} else if config.Auth.AWS != nil {
			authStrategy = infisical.AuthStrategy.AWS_IAM_MACHINE_IDENTITY
			if !config.Auth.AWS.IdentityId.IsNull() {
				identityId = config.Auth.AWS.IdentityId.ValueString()
			}
		} else if config.Auth.Token.ValueString() != "" {
			authStrategy = infisical.AuthStrategy.TOKEN_MACHINE_IDENTITY
			token = config.Auth.Token.ValueString()
		}
	}

	// strict env vars check:
	if authStrategy == "" {
		// ? note(daniel): this fix only works for token auth.
		// ? we currently don't have a way to identify if a user wants to use the different identity-id based auth strategies.
		// ? We should have a field for specifying the target auth strategy, like we do for the CLI (--method=aws-auth as an example)
		if envVarToken := os.Getenv(infisical.INFISICAL_TOKEN_NAME); envVarToken != "" {
			authStrategy = infisical.AuthStrategy.TOKEN_MACHINE_IDENTITY
			token = envVarToken
		}

	}

	client, err := infisical.NewClient(infisical.Config{
		HostURL:                 host,
		AuthStrategy:            authStrategy,
		ServiceToken:            serviceToken,
		ClientId:                clientId,
		ClientSecret:            clientSecret,
		IdentityId:              identityId,
		OidcTokenEnvName:        oidcTokenEnvName,
		Token:                   token,
		ServiceAccountToken:     serviceAccountToken,
		ServiceAccountTokenPath: serviceAccountTokenPath,
	})

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
		infisicalDatasource.NewIdentityDetailsDataSource,
		infisicalDatasource.NewKMSKeyDataSource,
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
		infisicalResource.NewIdentityTokenAuthResource,
		infisicalResource.NewIdentityTokenAuthTokenResource,
		infisicalResource.NewIntegrationGcpSecretManagerResource,
		infisicalResource.NewIntegrationAwsParameterStoreResource,
		infisicalResource.NewIntegrationAwsSecretsManagerResource,
		infisicalResource.NewIntegrationCircleCiResource,
		infisicalResource.NewIntegrationDatabricksResource,
		infisicalResource.NewSecretApprovalPolicyResource,
		infisicalResource.NewAccessApprovalPolicyResource,
		infisicalResource.NewProjectSecretImportResource,
		infisicalResource.NewGroupResource,
		appConnectionResource.NewAppConnectionGcpResource,
		appConnectionResource.NewAppConnectionAwsResource,
		appConnectionResource.NewAppConnectionAzureResource,
		appConnectionResource.NewAppConnectionAzureKeyVaultResource,
		appConnectionResource.NewAppConnection1PasswordResource,
		appConnectionResource.NewAppConnectionAzureAppConfigurationResource,
		appConnectionResource.NewAppConnectionRenderResource,
		appConnectionResource.NewAppConnectionMySqlResource,
		appConnectionResource.NewAppConnectionMsSqlResource,
		appConnectionResource.NewAppConnectionPostgresResource,
		appConnectionResource.NewAppConnectionOracleDbResource,
		appConnectionResource.NewAppConnectionBitbucketResource,
		appConnectionResource.NewAppConnectionDatabricksResource,
		appConnectionResource.NewAppConnectionCloudflareResource,
		appConnectionResource.NewAppConnectionSupabaseResource,
		appConnectionResource.NewAppConnectionFlyioResource,
		appConnectionResource.NewAppConnectionLdapResource,
		appConnectionResource.NewAppConnectionGitlabResource,
		secretSyncResource.NewSecretSyncGcpSecretManagerResource,
		secretSyncResource.NewSecretSyncAzureAppConfigurationResource,
		secretSyncResource.NewSecretSyncAzureKeyVaultResource,
		secretSyncResource.NewSecretSyncAwsParameterStoreResource,
		secretSyncResource.NewSecretSyncAwsSecretsManagerResource,
		secretSyncResource.NewSecretSyncGithubResource,
		secretSyncResource.NewSecretSync1PasswordResource,
		secretSyncResource.NewSecretSyncAzureDevOpsResource,
		secretSyncResource.NewSecretSyncRenderResource,
		secretSyncResource.NewSecretSyncBitbucketResource,
		secretSyncResource.NewSecretSyncDatabricksResource,
		secretSyncResource.NewSecretSyncCloudflareWorkersResource,
		secretSyncResource.NewSecretSyncCloudflarePagesResource,
		secretSyncResource.NewSecretSyncSupabaseResource,
		secretSyncResource.NewSecretSyncFlyioResource,
		secretSyncResource.NewSecretSyncGitlabResource,
		dynamicSecretResource.NewDynamicSecretSqlDatabaseResource,
		dynamicSecretResource.NewDynamicSecretAwsIamResource,
		dynamicSecretResource.NewDynamicSecretKubernetesResource,
		dynamicSecretResource.NewDynamicSecretMongoAtlasResource,
		dynamicSecretResource.NewDynamicSecretMongoDbResource,
		secretRotationResource.NewSecretRotationMySqlCredentialsResource,
		secretRotationResource.NewSecretRotationMsSqlCredentialsResource,
		secretRotationResource.NewSecretRotationPostgresCredentialsResource,
		secretRotationResource.NewSecretRotationOracleDbCredentialsResource,
		secretRotationResource.NewSecretRotationAzureClientSecretResource,
		secretRotationResource.NewSecretRotationAwsIamUserSecretResource,
		secretRotationResource.NewSecretRotationLdapPasswordResource,
		infisicalResource.NewProjectTemplateResource,
		infisicalResource.NewKMSKeyResource,
		externalKmsResource.NewExternalKmsAwsResource,
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
