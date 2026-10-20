package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AppConnectionSshCredentialsModel struct {
	Host       types.String `tfsdk:"host"`
	Port       types.Int32  `tfsdk:"port"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	PrivateKey types.String `tfsdk:"private_key"`
	Passphrase types.String `tfsdk:"passphrase"`
}

const SshAppConnectionPasswordMethod = "password"
const SshAppConnectionSshKeyMethod = "ssh-key"
const sshAppConnectionDefaultPort = 22

func buildSshCredentials(credentials AppConnectionSshCredentialsModel, method string, errorSummary string) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	credentialsConfig := make(map[string]any)

	if credentials.Host.IsNull() || credentials.Host.ValueString() == "" {
		diags.AddError(errorSummary, "Host field must be defined for all methods")
		return nil, diags
	}

	if credentials.Username.IsNull() || credentials.Username.ValueString() == "" {
		diags.AddError(errorSummary, "Username field must be defined for all methods")
		return nil, diags
	}

	port := int32(sshAppConnectionDefaultPort)
	if !credentials.Port.IsNull() && !credentials.Port.IsUnknown() {
		port = credentials.Port.ValueInt32()
	}

	credentialsConfig["host"] = credentials.Host.ValueString()
	credentialsConfig["port"] = port
	credentialsConfig["username"] = credentials.Username.ValueString()

	if method == SshAppConnectionPasswordMethod {
		if credentials.Password.IsNull() || credentials.Password.ValueString() == "" {
			diags.AddError(errorSummary, "Password field must be defined in password method")
			return nil, diags
		}

		credentialsConfig["password"] = credentials.Password.ValueString()
	} else {
		if credentials.PrivateKey.IsNull() || credentials.PrivateKey.ValueString() == "" {
			diags.AddError(errorSummary, "Private key field must be defined in ssh-key method")
			return nil, diags
		}

		credentialsConfig["privateKey"] = credentials.PrivateKey.ValueString()

		if !credentials.Passphrase.IsNull() && credentials.Passphrase.ValueString() != "" {
			credentialsConfig["passphrase"] = credentials.Passphrase.ValueString()
		}
	}

	return credentialsConfig, diags
}

func NewAppConnectionSshResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppSSH,
		AppConnectionName: "SSH",
		ResourceTypeName:  "_app_connection_ssh",
		SupportsGateway:   true,
		AllowedMethods:    []string{SshAppConnectionPasswordMethod, SshAppConnectionSshKeyMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname or IP address of the SSH server.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(sshAppConnectionDefaultPort),
				Description: "The port of the SSH server.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to authenticate to the SSH server with.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The password to authenticate to the SSH server with, required for the `password` method.",
			},
			"private_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The PEM-encoded SSH private key to authenticate with, required for the `ssh-key` method.",
			},
			"passphrase": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The passphrase protecting the SSH private key, used only by the `ssh-key` method.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentials AppConnectionSshCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return buildSshCredentials(credentials, plan.Method.ValueString(), "Unable to create SSH app connection")
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentialsFromPlan AppConnectionSshCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionSshCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			credentials := credentialsFromPlan
			if credentialsFromPlan.Host.IsUnknown() {
				credentials.Host = credentialsFromState.Host
			}
			if credentialsFromPlan.Port.IsUnknown() {
				credentials.Port = credentialsFromState.Port
			}
			if credentialsFromPlan.Username.IsUnknown() {
				credentials.Username = credentialsFromState.Username
			}
			if credentialsFromPlan.Password.IsUnknown() {
				credentials.Password = credentialsFromState.Password
			}
			if credentialsFromPlan.PrivateKey.IsUnknown() {
				credentials.PrivateKey = credentialsFromState.PrivateKey
			}
			if credentialsFromPlan.Passphrase.IsUnknown() {
				credentials.Passphrase = credentialsFromState.Passphrase
			}

			return buildSshCredentials(credentials, plan.Method.ValueString(), "Unable to update SSH app connection")
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"host":        types.StringNull(),
				"port":        types.Int32Null(),
				"username":    types.StringNull(),
				"password":    types.StringNull(),
				"private_key": types.StringNull(),
				"passphrase":  types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"host":        types.StringType,
				"port":        types.Int32Type,
				"username":    types.StringType,
				"password":    types.StringType,
				"private_key": types.StringType,
				"passphrase":  types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
