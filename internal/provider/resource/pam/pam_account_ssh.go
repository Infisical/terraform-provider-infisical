package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	pamSshAuthMethodPassword    = "password"
	pamSshAuthMethodPublicKey   = "public-key"
	pamSshAuthMethodCertificate = "certificate"
)

type PamAccountSshConnectionDetailsModel struct {
	Host types.String `tfsdk:"host"`
	Port types.Int64  `tfsdk:"port"`
}

type PamAccountSshCredentialsModel struct {
	AuthMethod types.String `tfsdk:"auth_method"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	PrivateKey types.String `tfsdk:"private_key"`
}

func NewPamAccountSshResource() resource.Resource {
	return &PamAccountBaseResource{
		AccountType:      "ssh",
		AccountTypeName:  "SSH",
		ResourceTypeName: "_pam_account_ssh",
		ConnectionDetailsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The hostname or IP address of the SSH server.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"port": schema.Int64Attribute{
				Description: "The port of the SSH server. Defaults to 22.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(22),
				Validators:  []validator.Int64{int64validator.Between(1, 65535)},
			},
		},
		CredentialsAttributes: map[string]schema.Attribute{
			"auth_method": schema.StringAttribute{
				Description: "How the account authenticates: `password`, `public-key`, or `certificate`. With `certificate`, the server must trust the account's Infisical SSH CA: once the account exists, get its public key from `GET /api/v1/pam/accounts/{id}/ssh-ca-public-key` and add it to the server's `TrustedUserCAKeys`. Creation doesn't check this trust, so sessions fail until it is set up.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf(pamSshAuthMethodPassword, pamSshAuthMethodPublicKey, pamSshAuthMethodCertificate)},
			},
			"username": schema.StringAttribute{
				Description: "The username to log in as.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"password": schema.StringAttribute{
				Description: "The password, for the `password` auth method.",
				Optional:    true,
				Sensitive:   true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"private_key": schema.StringAttribute{
				Description: "The PEM-encoded private key, for the `public-key` auth method.",
				Optional:    true,
				Sensitive:   true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 5000)},
			},
		},

		ConnectionDetailsToAPI: func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountSshConnectionDetailsModel
			diags := connectionDetails.As(ctx, &model, basetypes.ObjectAsOptions{})
			return map[string]any{
				"host": model.Host.ValueString(),
				"port": model.Port.ValueInt64(),
			}, diags
		},

		ConnectionDetailsFromAPI: func(connectionDetails map[string]any) any {
			return PamAccountSshConnectionDetailsModel{
				Host: apiString(connectionDetails, "host"),
				Port: apiInt64(connectionDetails, "port"),
			}
		},

		CredentialsToAPI: func(ctx context.Context, credentials types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountSshCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{
				"authMethod": model.AuthMethod.ValueString(),
				"username":   model.Username.ValueString(),
			}
			setString(values, "password", model.Password)
			setString(values, "privateKey", model.PrivateKey)
			return values, diags
		},

		CredentialsFromAPI: func(_ context.Context, credentials map[string]any, prior types.Object) (any, diag.Diagnostics) {
			return PamAccountSshCredentialsModel{
				AuthMethod: apiString(credentials, "authMethod"),
				Username:   apiString(credentials, "username"),
				Password:   priorString(prior, "password"),
				PrivateKey: priorString(prior, "private_key"),
			}, nil
		},

		ValidateTypeConfig: func(ctx context.Context, _ types.Object, credentials types.Object) diag.Diagnostics {
			var model PamAccountSshCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			if diags.HasError() || model.AuthMethod.IsUnknown() {
				return diags
			}

			method := model.AuthMethod.ValueString()
			if !model.Password.IsNull() && method != pamSshAuthMethodPassword {
				diags.AddAttributeError(path.Root("credentials").AtName("password"), "Unexpected credential",
					"`password` is only used by the `password` auth method.")
			}
			if !model.PrivateKey.IsNull() && method != pamSshAuthMethodPublicKey {
				diags.AddAttributeError(path.Root("credentials").AtName("private_key"), "Unexpected credential",
					"`private_key` is only used by the `public-key` auth method.")
			}
			return diags
		},
	}
}
