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
var _ datasource.DataSource = &OrganizationDataSource{}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

// OrganizationDataSource defines the data source implementation.
type OrganizationDataSource struct {
	client *infisical.Client
}

// OrganizationDataSourceModel describes the data source data model.
type OrganizationDataSourceModel struct {
	Slug types.String `tfsdk:"slug"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *OrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an organization by slug to obtain its ID. The slug can be that of the organization the provider is authenticated to, or of any sub-organization under the caller's root organization. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the organization.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Computed:    true,
			},
		},
	}
}

func (d *OrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch organization",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data OrganizationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	organization, err := d.client.GetOrganizationBySlug(data.Slug.ValueString())
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Organization not found",
				fmt.Sprintf("No organization was found with slug %s. Only the organization the machine identity is authenticated to and sub-organizations under its root organization can be resolved.", data.Slug.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the organization",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(organization.ID)
	data.Name = types.StringValue(organization.Name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
