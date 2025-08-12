package datasource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IdentityDetailsDataSource{}

func NewIdentityDetailsDataSource() datasource.DataSource {
	return &IdentityDetailsDataSource{}
}

// IdentityDetailsDataSource defines the data source implementation.
type IdentityDetailsDataSource struct {
	client *infisical.Client
}

// OrganizationModel represents the organization nested object
type OrganizationModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Slug types.String `tfsdk:"slug"`
}

// IdentityDetailsDataSourceModel describes the data source data model.
type IdentityDetailsDataSourceModel struct {
	Organization types.Object `tfsdk:"organization"`
}

func (d *IdentityDetailsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_details"
}

func (d *IdentityDetailsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Infisical identity details.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.SingleNestedAttribute{
				Description: "Organization details",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The ID of the organization",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the organization",
						Computed:    true,
					},
					"slug": schema.StringAttribute{
						Description: "The slug of the organization",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *IdentityDetailsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *IdentityDetailsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch identity details",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentityDetailsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	identityDetails, err := d.client.GetIdentityDetails()
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the identity details",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
	}

	organizationData := OrganizationModel{
		ID:   types.StringValue(identityDetails.IdentityDetails.OrganizationID),
		Name: types.StringValue(identityDetails.IdentityDetails.OrganizationName),
		Slug: types.StringValue(identityDetails.IdentityDetails.OrganizationSlug),
	}

	organizationValue, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"slug": types.StringType,
	}, organizationData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Organization = organizationValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
