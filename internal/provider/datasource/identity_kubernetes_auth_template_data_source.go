package datasource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = &IdentityKubernetesAuthTemplateDataSource{}
	_ datasource.DataSourceWithConfigValidators = &IdentityKubernetesAuthTemplateDataSource{}
)

func NewIdentityKubernetesAuthTemplateDataSource() datasource.DataSource {
	return &IdentityKubernetesAuthTemplateDataSource{}
}

type IdentityKubernetesAuthTemplateDataSource struct {
	client *infisical.Client
}

type IdentityKubernetesAuthTemplateDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	TokenReviewerMode    types.String `tfsdk:"token_reviewer_mode"`
	KubernetesHost       types.String `tfsdk:"kubernetes_host"`
	CaCertificate        types.String `tfsdk:"kubernetes_ca_certificate"`
	VerifyTlsCertificate types.Bool   `tfsdk:"verify_tls_certificate"`
	HasTokenReviewerJWT  types.Bool   `tfsdk:"has_token_reviewer_jwt"`
	GatewayID            types.String `tfsdk:"gateway_id"`
	GatewayPoolID        types.String `tfsdk:"gateway_pool_id"`
	AllowedAudience      types.String `tfsdk:"allowed_audience"`
}

func (d *IdentityKubernetesAuthTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_kubernetes_auth_template"
}

func (d *IdentityKubernetesAuthTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Kubernetes auth template by ID or name, for use with the `template_id` attribute on `infisical_identity_kubernetes_auth`. Exactly one of `id` or `name` must be set. The token reviewer JWT is write-only and never returned. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the auth template.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{infisicaltf.UuidValidator},
			},
			"name": schema.StringAttribute{
				Description: "The name of the auth template. Names are not unique, so a lookup by a name that several Kubernetes templates share fails and must use `id` instead.",
				Optional:    true,
				Computed:    true,
			},
			"token_reviewer_mode": schema.StringAttribute{
				Description: "Who performs the TokenReview: `api` or `gateway`.",
				Computed:    true,
			},
			"kubernetes_host": schema.StringAttribute{
				Description: "The host string, host:port pair, or URL to the base of the Kubernetes API server.",
				Computed:    true,
			},
			"kubernetes_ca_certificate": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate used to validate the Kubernetes API server's TLS certificate.",
				Computed:    true,
			},
			"verify_tls_certificate": schema.BoolAttribute{
				Description: "Whether the Kubernetes API server's TLS certificate is verified.",
				Computed:    true,
			},
			"has_token_reviewer_jwt": schema.BoolAttribute{
				Description: "Whether Infisical holds a token reviewer JWT for this template.",
				Computed:    true,
			},
			"gateway_id": schema.StringAttribute{
				Description: "The ID of the gateway Kubernetes API requests are routed through.",
				Computed:    true,
			},
			"gateway_pool_id": schema.StringAttribute{
				Description: "The ID of the gateway pool Kubernetes API requests are routed through.",
				Computed:    true,
			},
			"allowed_audience": schema.StringAttribute{
				Description: "The audience claim that service account JWTs must carry to authenticate.",
				Computed:    true,
			},
		},
	}
}

func (d *IdentityKubernetesAuthTemplateDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *IdentityKubernetesAuthTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *infisical.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *IdentityKubernetesAuthTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch identity kubernetes auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentityKubernetesAuthTemplateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var template infisical.IdentityKubernetesAuthTemplate
	var err error
	lookup := ""
	if !data.ID.IsNull() {
		lookup = "ID " + data.ID.ValueString()
		template, err = d.client.GetIdentityKubernetesAuthTemplate(data.ID.ValueString())
	} else {
		lookup = fmt.Sprintf("name %q", data.Name.ValueString())
		template, err = d.client.GetIdentityKubernetesAuthTemplateByName(data.Name.ValueString())
	}

	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Identity kubernetes auth template not found",
				"No Kubernetes auth template was found with "+lookup+" in the machine identity's organization.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the identity kubernetes auth template",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	fields := template.TemplateFields
	data.ID = types.StringValue(template.ID)
	data.Name = types.StringValue(template.Name)
	data.TokenReviewerMode = types.StringValue(fields.TokenReviewMode)
	data.KubernetesHost = types.StringPointerValue(nonEmptyStringPtr(fields.KubernetesHost))
	data.CaCertificate = types.StringValue(fields.CaCert)
	if fields.VerifyTlsCertificate != nil {
		data.VerifyTlsCertificate = types.BoolValue(*fields.VerifyTlsCertificate)
	} else {
		data.VerifyTlsCertificate = types.BoolValue(fields.CaCert != "")
	}
	data.HasTokenReviewerJWT = types.BoolValue(fields.HasTokenReviewerJwt)
	data.GatewayID = types.StringPointerValue(nonEmptyStringPtr(fields.GatewayID))
	data.GatewayPoolID = types.StringPointerValue(nonEmptyStringPtr(fields.GatewayPoolID))
	data.AllowedAudience = types.StringValue(fields.AllowedAudience)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func nonEmptyStringPtr(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}
