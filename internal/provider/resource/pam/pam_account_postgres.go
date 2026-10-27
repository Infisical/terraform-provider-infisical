package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	pamPostgresAuthMethodPassword = "password"
	pamPostgresAuthMethodAwsIam   = "aws-iam"
)

type PamAccountPostgresConnectionDetailsModel struct {
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int64  `tfsdk:"port"`
	Database              types.String `tfsdk:"database"`
	SslEnabled            types.Bool   `tfsdk:"ssl_enabled"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

type PamAccountPostgresCredentialsModel struct {
	AuthMethod types.String `tfsdk:"auth_method"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	AwsRegion  types.String `tfsdk:"aws_region"`
	RoleArn    types.String `tfsdk:"role_arn"`
}

func NewPamAccountPostgresResource() resource.Resource {
	return &PamAccountBaseResource{
		AccountType:      "postgres",
		AccountTypeName:  "PostgreSQL",
		ResourceTypeName: "_pam_account_postgres",
		SupportsRotation: true,
		ConnectionDetailsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The hostname or IP address of the database server.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"port": schema.Int64Attribute{
				Description: "The port of the database server. Defaults to 5432.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5432),
				Validators:  []validator.Int64{int64validator.Between(1, 65535)},
			},
			"database": schema.StringAttribute{
				Description: "The name of the database to connect to.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"ssl_enabled": schema.BoolAttribute{
				Description: "Whether to connect over TLS. Must be true for the `aws-iam` auth method. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Description: "Whether to reject a server certificate that isn't trusted. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ssl_certificate": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate to verify the server with.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		CredentialsAttributes: map[string]schema.Attribute{
			"auth_method": schema.StringAttribute{
				Description: "How the account authenticates: `password`, or `aws-iam` for RDS and Aurora IAM authentication. Defaults to `password`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(pamPostgresAuthMethodPassword),
				Validators:  []validator.String{stringvalidator.OneOf(pamPostgresAuthMethodPassword, pamPostgresAuthMethodAwsIam)},
			},
			"username": schema.StringAttribute{
				Description: "The database user.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 63)},
			},
			"password": schema.StringAttribute{
				Description: "The password, for the `password` auth method.",
				Optional:    true,
				Sensitive:   true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"aws_region": schema.StringAttribute{
				Description: "The AWS region of the RDS or Aurora instance, for the `aws-iam` auth method.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 32)},
			},
			"role_arn": schema.StringAttribute{
				Description: "The ARN of the IAM role Infisical assumes to generate the auth token, for the `aws-iam` auth method. It needs `rds-db:connect` and must trust Infisical with your organization ID as the external ID.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 2048)},
			},
		},

		ConnectionDetailsToAPI: func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountPostgresConnectionDetailsModel
			diags := connectionDetails.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{
				"host":                  model.Host.ValueString(),
				"port":                  model.Port.ValueInt64(),
				"database":              model.Database.ValueString(),
				"sslEnabled":            model.SslEnabled.ValueBool(),
				"sslRejectUnauthorized": model.SslRejectUnauthorized.ValueBool(),
			}
			setString(values, "sslCertificate", model.SslCertificate)
			return values, diags
		},

		ConnectionDetailsFromAPI: func(connectionDetails map[string]any) any {
			return PamAccountPostgresConnectionDetailsModel{
				Host:                  apiString(connectionDetails, "host"),
				Port:                  apiInt64(connectionDetails, "port"),
				Database:              apiString(connectionDetails, "database"),
				SslEnabled:            apiBool(connectionDetails, "sslEnabled"),
				SslRejectUnauthorized: apiBool(connectionDetails, "sslRejectUnauthorized"),
				SslCertificate:        apiString(connectionDetails, "sslCertificate"),
			}
		},

		CredentialsToAPI: func(ctx context.Context, credentials types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountPostgresCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{
				"authMethod": model.AuthMethod.ValueString(),
				"username":   model.Username.ValueString(),
			}
			setString(values, "password", model.Password)
			setString(values, "awsRegion", model.AwsRegion)
			setString(values, "roleArn", model.RoleArn)
			return values, diags
		},

		CredentialsFromAPI: func(_ context.Context, credentials map[string]any, prior types.Object) (any, diag.Diagnostics) {
			authMethod := apiString(credentials, "authMethod")
			// Older accounts store no auth method and use a password.
			if authMethod.IsNull() {
				authMethod = types.StringValue(pamPostgresAuthMethodPassword)
			}
			return PamAccountPostgresCredentialsModel{
				AuthMethod: authMethod,
				Username:   apiString(credentials, "username"),
				Password:   priorString(prior, "password"),
				AwsRegion:  apiString(credentials, "awsRegion"),
				RoleArn:    apiString(credentials, "roleArn"),
			}, nil
		},

		ValidateTypeConfig: func(ctx context.Context, connectionDetails types.Object, credentials types.Object) diag.Diagnostics {
			var model PamAccountPostgresCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			if diags.HasError() || model.AuthMethod.IsUnknown() {
				return diags
			}

			credentialsPath := path.Root("credentials")
			if model.AuthMethod.ValueString() != pamPostgresAuthMethodAwsIam {
				for name, value := range map[string]types.String{"aws_region": model.AwsRegion, "role_arn": model.RoleArn} {
					if !value.IsNull() {
						diags.AddAttributeError(credentialsPath.AtName(name), "Unexpected credential",
							"`"+name+"` is only used by the `aws-iam` auth method.")
					}
				}
				return diags
			}

			if !model.Password.IsNull() {
				diags.AddAttributeError(credentialsPath.AtName("password"), "Unexpected credential",
					"`password` is only used by the `password` auth method.")
			}
			for name, value := range map[string]types.String{"aws_region": model.AwsRegion, "role_arn": model.RoleArn} {
				if value.IsNull() {
					diags.AddAttributeError(credentialsPath.AtName(name), "Missing credential",
						"`"+name+"` is required by the `aws-iam` auth method.")
				}
			}

			if connectionDetails.IsNull() {
				return diags
			}
			var connection PamAccountPostgresConnectionDetailsModel
			diags.Append(connectionDetails.As(ctx, &connection, basetypes.ObjectAsOptions{})...)
			if !connection.SslEnabled.IsNull() && !connection.SslEnabled.IsUnknown() && !connection.SslEnabled.ValueBool() {
				diags.AddAttributeError(path.Root("connection_details").AtName("ssl_enabled"), "TLS required",
					"The `aws-iam` auth method needs an encrypted connection, so `ssl_enabled` must be true.")
			}
			return diags
		},
	}
}
