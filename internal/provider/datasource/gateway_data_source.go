package datasource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GatewayDataSource{}

func NewGatewayDataSource() datasource.DataSource {
	return &GatewayDataSource{}
}

// GatewayDataSource defines the data source implementation.
type GatewayDataSource struct {
	client *infisical.Client
}

// GatewayDataSourceModel describes the data source data model.
type GatewayDataSourceModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

func (d *GatewayDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (d *GatewayDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a gateway by name to obtain its ID, for use with the `gateway_id` attribute on dynamic secrets and app connections. The gateway is resolved within the machine identity's organization. Only Machine Identity authentication is supported for this data source, and the identity must have permission to read gateways.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the gateway.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the gateway.",
				Computed:    true,
			},
		},
	}
}

func (d *GatewayDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *GatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch gateway",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data GatewayDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	gateway, err := d.client.GetGatewayByName(data.Name.ValueString())
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Gateway not found",
				fmt.Sprintf("No gateway was found with name %s in the machine identity's organization.", data.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the gateway",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(gateway.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
