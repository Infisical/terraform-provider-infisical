package datasource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &InstanceDetailsDataSource{}

func NewInstanceDetailsDataSource() datasource.DataSource {
	return &InstanceDetailsDataSource{}
}

// InstanceDetailsDataSource defines the data source implementation.
type InstanceDetailsDataSource struct {
	client *infisical.Client
}

// InstanceDetailsDataSourceModel describes the data source data model.
type InstanceDetailsDataSourceModel struct {
	AwsAccountId types.String `tfsdk:"aws_account_id"`
}

func (d *InstanceDetailsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_details"
}

func (d *InstanceDetailsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read details about the Infisical instance the provider is connected to, such as the AWS account ID to trust in an IAM role's trust policy. Works with any authentication method.",
		Attributes: map[string]schema.Attribute{
			"aws_account_id": schema.StringAttribute{
				Description: "The AWS account ID Infisical acts as when it assumes an IAM role in your AWS account, for example for an AWS App Connection. Name this account as the trusted principal in the role's trust policy. Null when the instance has no AWS identity, or when the Infisical server is too old to report it.",
				Computed:    true,
			},
		},
	}
}

func (d *InstanceDetailsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read has no machine identity check because GET /api/status is public.
func (d *InstanceDetailsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceDetailsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, err := d.client.GetInstanceStatus()
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the instance details",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.AwsAccountId = types.StringPointerValue(status.AwsAccountId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
