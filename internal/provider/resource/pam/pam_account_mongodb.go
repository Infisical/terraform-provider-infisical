package resource

import (
	"context"
	"regexp"
	"terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PamAccountMongoDBConnectionDetailsModel struct {
	ConnectionString      customtypes.MongoDBConnectionStringValue `tfsdk:"connection_string"`
	Database              types.String                             `tfsdk:"database"`
	SslEnabled            types.Bool                               `tfsdk:"ssl_enabled"`
	SslRejectUnauthorized types.Bool                               `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String                             `tfsdk:"ssl_certificate"`
}

type PamAccountMongoDBCredentialsModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type mongoDBConnectionStringValidator struct{}

func (mongoDBConnectionStringValidator) Description(_ context.Context) string {
	return "must be a mongodb:// or mongodb+srv:// URI with no credentials or database"
}

func (v mongoDBConnectionStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (mongoDBConnectionStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := customtypes.ValidateMongoDBConnectionString(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid connection string", "The connection string "+err.Error()+".")
	}
}

// mongoDBConnectionStringPlanModifier keeps the stored connection string when the configured one
// normalizes to it. After an import, state holds the API's form, and plan compares it with the
// configured one without semantic equality.
type mongoDBConnectionStringPlanModifier struct{}

func (mongoDBConnectionStringPlanModifier) Description(_ context.Context) string {
	return "keeps the stored connection string when it only differs from the configured one in how the API stores it"
}

func (m mongoDBConnectionStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (mongoDBConnectionStringPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if customtypes.NormalizeMongoDBConnectionString(req.ConfigValue.ValueString()) == customtypes.NormalizeMongoDBConnectionString(req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

func NewPamAccountMongoDBResource() resource.Resource {
	return &PamAccountBaseResource{
		AccountType:      "mongodb",
		AccountTypeName:  "MongoDB",
		ResourceTypeName: "_pam_account_mongodb",
		ConnectionDetailsAttributes: map[string]schema.Attribute{
			"connection_string": schema.StringAttribute{
				Description: "The `mongodb://` or `mongodb+srv://` URI of the deployment, with options such as `replicaSet` in its query string. It must not include credentials or a database.",
				Required:    true,
				CustomType:  customtypes.MongoDBConnectionStringType{},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
					mongoDBConnectionStringValidator{},
				},
				PlanModifiers: []planmodifier.String{mongoDBConnectionStringPlanModifier{}},
			},
			"database": schema.StringAttribute{
				Description: "The database to connect to, which is also where the user authenticates unless the connection string sets `authSource`. Defaults to `admin`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("admin"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), "must contain only letters, numbers, underscores and hyphens"),
				},
			},
			"ssl_enabled": schema.BoolAttribute{
				Description: "Whether to connect over TLS, which `mongodb+srv://` deployments such as Atlas need. Defaults to true. Set it to false for a server without TLS, since a mismatch makes the connection test time out.",
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
			"username": schema.StringAttribute{
				Description: "The database user.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"password": schema.StringAttribute{
				Description: "The password of the database user.",
				Optional:    true,
				Sensitive:   true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
		},

		ConnectionDetailsToAPI: func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountMongoDBConnectionDetailsModel
			diags := connectionDetails.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{
				"connectionString":      model.ConnectionString.ValueString(),
				"database":              model.Database.ValueString(),
				"sslEnabled":            model.SslEnabled.ValueBool(),
				"sslRejectUnauthorized": model.SslRejectUnauthorized.ValueBool(),
			}
			setString(values, "sslCertificate", model.SslCertificate)
			return values, diags
		},

		ConnectionDetailsFromAPI: func(connectionDetails map[string]any) any {
			connectionString := customtypes.NewMongoDBConnectionStringNull()
			if value := apiString(connectionDetails, "connectionString"); !value.IsNull() {
				connectionString = customtypes.NewMongoDBConnectionStringValue(value.ValueString())
			}
			return PamAccountMongoDBConnectionDetailsModel{
				ConnectionString:      connectionString,
				Database:              apiString(connectionDetails, "database"),
				SslEnabled:            apiBool(connectionDetails, "sslEnabled"),
				SslRejectUnauthorized: apiBool(connectionDetails, "sslRejectUnauthorized"),
				SslCertificate:        apiString(connectionDetails, "sslCertificate"),
			}
		},

		CredentialsToAPI: func(ctx context.Context, credentials types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountMongoDBCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{"username": model.Username.ValueString()}
			setString(values, "password", model.Password)
			return values, diags
		},

		CredentialsFromAPI: func(_ context.Context, credentials map[string]any, prior types.Object) (any, diag.Diagnostics) {
			return PamAccountMongoDBCredentialsModel{
				Username: apiString(credentials, "username"),
				Password: priorString(prior, "password"),
			}, nil
		},
	}
}
