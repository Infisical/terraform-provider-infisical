package datasource

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IdentityUniversalAuthDataSource{}

func NewIdentityUniversalAuthDataSource() datasource.DataSource {
	return &IdentityUniversalAuthDataSource{}
}

// IdentityUniversalAuthDataSource defines the data source implementation.
type IdentityUniversalAuthDataSource struct {
	client *infisical.Client
}

// IdentityUniversalAuthDataSourceModel describes the data source data model.
type IdentityUniversalAuthDataSourceModel struct {
	IdentityID              types.String `tfsdk:"identity_id"`
	ID                      types.String `tfsdk:"id"`
	ClientID                types.String `tfsdk:"client_id"`
	AccessTokenTTL          types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64  `tfsdk:"access_token_num_uses_limit"`
	ClientSecretTrustedIps  types.List   `tfsdk:"client_secret_trusted_ips"`
	AccessTokenTrustedIps   types.List   `tfsdk:"access_token_trusted_ips"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

// identityUniversalAuthTrustedIpAttrTypes is the attribute type map for a trusted IP object.
var identityUniversalAuthTrustedIpAttrTypes = map[string]attr.Type{
	"ip_address": types.StringType,
}

// IdentityUniversalAuthDataSourceTrustedIp represents a single trusted IP entry.
type IdentityUniversalAuthDataSourceTrustedIp struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

func (d *IdentityUniversalAuthDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_universal_auth"
}

func (d *IdentityUniversalAuthDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up the Universal Auth configuration for a machine identity by its identity ID. Returns null values if Universal Auth is not configured on the identity. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"identity_id": schema.StringAttribute{
				Description: "The ID of the identity to look up Universal Auth for.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the Universal Auth configuration. Null if Universal Auth is not configured.",
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID used for Universal Auth token exchange. Null if Universal Auth is not configured.",
				Computed:    true,
			},
			"access_token_ttl": schema.Int64Attribute{
				Description: "The lifetime for an access token in seconds. Null if Universal Auth is not configured.",
				Computed:    true,
			},
			"access_token_max_ttl": schema.Int64Attribute{
				Description: "The maximum lifetime for an access token in seconds. Null if Universal Auth is not configured.",
				Computed:    true,
			},
			"access_token_num_uses_limit": schema.Int64Attribute{
				Description: "The maximum number of times an access token can be used; 0 means unlimited. Null if Universal Auth is not configured.",
				Computed:    true,
			},
			"client_secret_trusted_ips": schema.ListNestedAttribute{
				Description: "A list of IPs or CIDR ranges that the Client Secret can be used from. Null if Universal Auth is not configured.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"access_token_trusted_ips": schema.ListNestedAttribute{
				Description: "A list of IPs or CIDR ranges that access tokens can be used from. Null if Universal Auth is not configured.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The ISO 8601 timestamp when the Universal Auth configuration was created. Null if Universal Auth is not configured.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The ISO 8601 timestamp when the Universal Auth configuration was last updated. Null if Universal Auth is not configured.",
				Computed:    true,
			},
		},
	}
}

func (d *IdentityUniversalAuthDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *IdentityUniversalAuthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch identity universal auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentityUniversalAuthDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ipObjectType := types.ObjectType{AttrTypes: identityUniversalAuthTrustedIpAttrTypes}

	ua, err := d.client.GetIdentityUniversalAuth(infisical.GetIdentityUniversalAuthRequest{
		IdentityID: data.IdentityID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			data.ID = types.StringNull()
			data.ClientID = types.StringNull()
			data.AccessTokenTTL = types.Int64Null()
			data.AccessTokenMaxTTL = types.Int64Null()
			data.AccessTokenNumUsesLimit = types.Int64Null()
			data.ClientSecretTrustedIps = types.ListNull(ipObjectType)
			data.AccessTokenTrustedIps = types.ListNull(ipObjectType)
			data.CreatedAt = types.StringNull()
			data.UpdatedAt = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the identity universal auth configuration",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(ua.ID)
	data.ClientID = types.StringValue(ua.ClientID)
	data.AccessTokenTTL = types.Int64Value(ua.AccessTokenTTL)
	data.AccessTokenMaxTTL = types.Int64Value(ua.AccessTokenMaxTTL)
	data.AccessTokenNumUsesLimit = types.Int64Value(ua.AccessTokenNumUsesLimit)
	data.CreatedAt = types.StringValue(ua.CreatedAt)
	data.UpdatedAt = types.StringValue(ua.UpdatedAt)

	// Convert trusted IP lists, appending CIDR prefix when present.
	clientSecretIps := make([]IdentityUniversalAuthDataSourceTrustedIp, len(ua.ClientSecretTrustedIps))
	for i, el := range ua.ClientSecretTrustedIps {
		if el.Prefix != nil {
			clientSecretIps[i] = IdentityUniversalAuthDataSourceTrustedIp{IpAddress: types.StringValue(el.IpAddress + "/" + strconv.Itoa(*el.Prefix))}
		} else {
			clientSecretIps[i] = IdentityUniversalAuthDataSourceTrustedIp{IpAddress: types.StringValue(el.IpAddress)}
		}
	}

	accessTokenIps := make([]IdentityUniversalAuthDataSourceTrustedIp, len(ua.AccessTokenTrustedIps))
	for i, el := range ua.AccessTokenTrustedIps {
		if el.Prefix != nil {
			accessTokenIps[i] = IdentityUniversalAuthDataSourceTrustedIp{IpAddress: types.StringValue(el.IpAddress + "/" + strconv.Itoa(*el.Prefix))}
		} else {
			accessTokenIps[i] = IdentityUniversalAuthDataSourceTrustedIp{IpAddress: types.StringValue(el.IpAddress)}
		}
	}

	clientSecretList, diags := types.ListValueFrom(ctx, ipObjectType, clientSecretIps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ClientSecretTrustedIps = clientSecretList

	accessTokenList, diags := types.ListValueFrom(ctx, ipObjectType, accessTokenIps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AccessTokenTrustedIps = accessTokenList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
