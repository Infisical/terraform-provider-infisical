package datasource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = &IdentityOidcAuthTemplateDataSource{}
	_ datasource.DataSourceWithConfigValidators = &IdentityOidcAuthTemplateDataSource{}
)

func NewIdentityOidcAuthTemplateDataSource() datasource.DataSource {
	return &IdentityOidcAuthTemplateDataSource{}
}

type IdentityOidcAuthTemplateDataSource struct {
	client *infisical.Client
}

type IdentityOidcAuthTemplateDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	OidcDiscoveryUrl types.String `tfsdk:"oidc_discovery_url"`
	BoundIssuer      types.String `tfsdk:"bound_issuer"`
	BoundAudiences   types.List   `tfsdk:"bound_audiences"`
	CaCertificate    types.String `tfsdk:"oidc_ca_certificate"`
}

func (d *IdentityOidcAuthTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_oidc_auth_template"
}

func (d *IdentityOidcAuthTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an OIDC auth template by ID or name, for use with the `template_id` attribute on `infisical_identity_oidc_auth`. Exactly one of `id` or `name` must be set. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the auth template.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{infisicaltf.UuidValidator},
			},
			"name": schema.StringAttribute{
				Description: "The name of the auth template. Names are not unique, so a lookup by a name that several OIDC templates share fails and must use `id` instead.",
				Optional:    true,
				Computed:    true,
			},
			"oidc_discovery_url": schema.StringAttribute{
				Description: "The URL used to retrieve the OpenID Connect configuration from the identity provider.",
				Computed:    true,
			},
			"bound_issuer": schema.StringAttribute{
				Description: "The unique identifier of the identity provider issuing the OIDC tokens.",
				Computed:    true,
			},
			"bound_audiences": schema.ListAttribute{
				Description: "The intended recipients that OIDC tokens must carry in their aud claim.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"oidc_ca_certificate": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate for establishing secure communication with the identity provider endpoints.",
				Computed:    true,
			},
		},
	}
}

func (d *IdentityOidcAuthTemplateDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *IdentityOidcAuthTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentityOidcAuthTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch identity oidc auth template",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentityOidcAuthTemplateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var template infisical.IdentityOidcAuthTemplate
	var err error
	lookup := ""
	if !data.ID.IsNull() {
		lookup = "ID " + data.ID.ValueString()
		template, err = d.client.GetIdentityOidcAuthTemplate(data.ID.ValueString())
	} else {
		lookup = fmt.Sprintf("name %q", data.Name.ValueString())
		template, err = d.client.GetIdentityOidcAuthTemplateByName(data.Name.ValueString())
	}

	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Identity oidc auth template not found",
				"No OIDC auth template was found with "+lookup+" in the machine identity's organization.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the identity oidc auth template",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	fields := template.TemplateFields
	data.ID = types.StringValue(template.ID)
	data.Name = types.StringValue(template.Name)
	data.OidcDiscoveryUrl = types.StringValue(fields.OidcDiscoveryUrl)
	data.BoundIssuer = types.StringValue(fields.BoundIssuer)
	data.CaCertificate = types.StringValue(fields.CaCert)

	boundAudiences, diags := types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(fields.BoundAudiences, ","))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.BoundAudiences = boundAudiences

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
