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
var _ datasource.DataSource = &GroupsDataSource{}

func NewGroupsDataSource() datasource.DataSource {
	return &GroupsDataSource{}
}

// GroupsDataSource defines the data source implementation.
type GroupsDataSource struct {
	client *infisical.Client
}

// ExampleDataSourceModel describes the data source data model.
type GroupsDataSourceModel struct {
	Groups types.List `tfsdk:"groups"`
}

type InfisicalGroupDetails struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	OrgID  types.String `tfsdk:"org_id"`
	Role   types.String `tfsdk:"role"`
	RoleId types.String `tfsdk:"role_id"`
}

func (d *GroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

func (d *GroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Infisical groups in your organization.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
				Description: "The groups list",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the group",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the group",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The organization ID of the group",
							Computed:    true,
						},
						"role": schema.StringAttribute{
							Description: "The role of the group in the organization",
							Computed:    true,
						},
						"role_id": schema.StringAttribute{
							Description: "The role ID of the group in the organization",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *GroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch groups",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data GroupsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.GetGroups()
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the groups",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
	}

	planGroups := make([]InfisicalGroupDetails, len(groups))
	for i, el := range groups {
		planGroups[i] = InfisicalGroupDetails{
			ID:     types.StringValue(el.ID),
			Name:   types.StringValue(el.Name),
			OrgID:  types.StringValue(el.OrgID),
			Role:   types.StringValue(el.Role),
			RoleId: types.StringValue(el.RoleId),
		}
	}

	stateGroups, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":      types.StringType,
			"name":    types.StringType,
			"org_id":  types.StringType,
			"role":    types.StringType,
			"role_id": types.StringType,
		},
	}, planGroups)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Groups = stateGroups

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
