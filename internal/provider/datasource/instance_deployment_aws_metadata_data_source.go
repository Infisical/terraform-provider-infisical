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
var _ datasource.DataSource = &InstanceDeploymentAwsMetadataDataSource{}

func NewInstanceDeploymentAwsMetadataDataSource() datasource.DataSource {
	return &InstanceDeploymentAwsMetadataDataSource{}
}

// InstanceDeploymentAwsMetadataDataSource defines the data source implementation.
type InstanceDeploymentAwsMetadataDataSource struct {
	client *infisical.Client
}

// InstanceDeploymentAwsMetadataDataSourceModel describes the data source data model.
type InstanceDeploymentAwsMetadataDataSourceModel struct {
	AccountId types.String `tfsdk:"account_id"`
}

func (d *InstanceDeploymentAwsMetadataDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_deployment_aws_metadata"
}

func (d *InstanceDeploymentAwsMetadataDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read details about the AWS account the Infisical instance runs as, such as the account ID to trust in an IAM role's trust policy. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Description: "The AWS account ID Infisical acts as when it assumes an IAM role in your AWS account, for example for an AWS App Connection. Name this account as the trusted principal in the role's trust policy. Null when the instance has no AWS identity.",
				Computed:    true,
			},
		},
	}
}

func (d *InstanceDeploymentAwsMetadataDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceDeploymentAwsMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch the instance's AWS deployment metadata",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data InstanceDeploymentAwsMetadataDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata, err := d.client.GetInstanceDeploymentAwsMetadata()
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"AWS deployment metadata is not available on this Infisical server",
				"The Infisical server doesn't have the endpoint for the instance's AWS deployment metadata. Upgrade Infisical to a version that includes it.",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the instance's AWS deployment metadata",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.AccountId = types.StringPointerValue(metadata.AccountId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
