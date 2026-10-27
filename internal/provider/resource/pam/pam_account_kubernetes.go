package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	pamKubernetesAuthMethodServiceAccountToken = "service-account-token"
	pamKubernetesAuthMethodGateway             = "gateway-kubernetes-auth"
)

type PamAccountKubernetesConnectionDetailsModel struct {
	URL                   types.String `tfsdk:"url"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

type PamAccountKubernetesCredentialsModel struct {
	AuthMethod          types.String `tfsdk:"auth_method"`
	ServiceAccountToken types.String `tfsdk:"service_account_token"`
	Namespace           types.String `tfsdk:"namespace"`
	ServiceAccountName  types.String `tfsdk:"service_account_name"`
}

func NewPamAccountKubernetesResource() resource.Resource {
	return &PamAccountBaseResource{
		AccountType:      "kubernetes",
		AccountTypeName:  "Kubernetes",
		ResourceTypeName: "_pam_account_kubernetes",
		ConnectionDetailsAttributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The URL of the Kubernetes API server.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 500)},
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Description: "Whether to reject an API server certificate that isn't trusted. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ssl_certificate": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate to verify the API server with.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		CredentialsAttributes: map[string]schema.Attribute{
			"auth_method": schema.StringAttribute{
				Description: "How the account authenticates: `service-account-token` with a stored token, or `gateway-kubernetes-auth` through the gateway as `service_account_name` in `namespace`.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf(pamKubernetesAuthMethodServiceAccountToken, pamKubernetesAuthMethodGateway)},
			},
			"service_account_token": schema.StringAttribute{
				Description: "The service account token, for the `service-account-token` auth method.",
				Optional:    true,
				Sensitive:   true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 10000)},
			},
			"namespace": schema.StringAttribute{
				Description: "The namespace of the service account, for the `gateway-kubernetes-auth` auth method.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 63)},
			},
			"service_account_name": schema.StringAttribute{
				Description: "The name of the service account, for the `gateway-kubernetes-auth` auth method.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 253)},
			},
		},

		ConnectionDetailsToAPI: func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountKubernetesConnectionDetailsModel
			diags := connectionDetails.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{
				"url":                   model.URL.ValueString(),
				"sslRejectUnauthorized": model.SslRejectUnauthorized.ValueBool(),
			}
			setString(values, "sslCertificate", model.SslCertificate)
			return values, diags
		},

		ConnectionDetailsFromAPI: func(connectionDetails map[string]any) any {
			return PamAccountKubernetesConnectionDetailsModel{
				URL:                   apiString(connectionDetails, "url"),
				SslRejectUnauthorized: apiBool(connectionDetails, "sslRejectUnauthorized"),
				SslCertificate:        apiString(connectionDetails, "sslCertificate"),
			}
		},

		CredentialsToAPI: func(ctx context.Context, credentials types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountKubernetesCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{"authMethod": model.AuthMethod.ValueString()}
			setString(values, "serviceAccountToken", model.ServiceAccountToken)
			setString(values, "namespace", model.Namespace)
			setString(values, "serviceAccountName", model.ServiceAccountName)
			return values, diags
		},

		CredentialsFromAPI: func(_ context.Context, credentials map[string]any, prior types.Object) (any, diag.Diagnostics) {
			return PamAccountKubernetesCredentialsModel{
				AuthMethod:          apiString(credentials, "authMethod"),
				ServiceAccountToken: priorString(prior, "service_account_token"),
				Namespace:           apiString(credentials, "namespace"),
				ServiceAccountName:  apiString(credentials, "serviceAccountName"),
			}, nil
		},

		ValidateTypeConfig: func(ctx context.Context, _ types.Object, credentials types.Object) diag.Diagnostics {
			var model PamAccountKubernetesCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			if diags.HasError() || model.AuthMethod.IsUnknown() {
				return diags
			}

			credentialsPath := path.Root("credentials")
			gatewayFields := map[string]types.String{"namespace": model.Namespace, "service_account_name": model.ServiceAccountName}

			if model.AuthMethod.ValueString() == pamKubernetesAuthMethodServiceAccountToken {
				if model.ServiceAccountToken.IsNull() {
					diags.AddAttributeError(credentialsPath.AtName("service_account_token"), "Missing credential",
						"`service_account_token` is required by the `service-account-token` auth method.")
				}
				for name, value := range gatewayFields {
					if !value.IsNull() {
						diags.AddAttributeError(credentialsPath.AtName(name), "Unexpected credential",
							"`"+name+"` is only used by the `gateway-kubernetes-auth` auth method.")
					}
				}
				return diags
			}

			if !model.ServiceAccountToken.IsNull() {
				diags.AddAttributeError(credentialsPath.AtName("service_account_token"), "Unexpected credential",
					"`service_account_token` is only used by the `service-account-token` auth method.")
			}
			for name, value := range gatewayFields {
				if value.IsNull() {
					diags.AddAttributeError(credentialsPath.AtName(name), "Missing credential",
						"`"+name+"` is required by the `gateway-kubernetes-auth` auth method.")
				}
			}
			return diags
		},
	}
}
