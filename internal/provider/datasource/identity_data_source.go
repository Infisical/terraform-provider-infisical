package datasource

import (
	"context"
	"errors"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IdentityDataSource{}

func NewIdentityDataSource() datasource.DataSource {
	return &IdentityDataSource{}
}

// IdentityDataSource defines the data source implementation.
type IdentityDataSource struct {
	client *infisical.Client
}

// IdentityDataSourceMetaEntry describes a single metadata key/value pair.
type IdentityDataSourceMetaEntry struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// identityMetadataAttrTypes is the attribute type map for a single metadata
// object. Shared by the schema and types.ListValueFrom to keep them in sync.
var identityMetadataAttrTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

// IdentityDataSourceModel describes the data source data model.
type IdentityDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	HasDeleteProtection types.Bool   `tfsdk:"has_delete_protection"`
	AuthModes           types.List   `tfsdk:"auth_modes"`
	Role                types.String `tfsdk:"role"`
	CustomRoleID        types.String `tfsdk:"custom_role_id"`
	OrgID               types.String `tfsdk:"org_id"`
	Metadata            types.List   `tfsdk:"metadata"`
}

func (d *IdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (d *IdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an Infisical identity by its ID. Returns the identity's details including its name, organization role, auth methods, and metadata. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the identity to look up",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the identity",
				Computed:    true,
			},
			"has_delete_protection": schema.BoolAttribute{
				Description: "Whether the identity has delete protection enabled",
				Computed:    true,
			},
			"auth_modes": schema.ListAttribute{
				Description: "The authentication methods configured on the identity",
				Computed:    true,
				ElementType: types.StringType,
			},
			"role": schema.StringAttribute{
				Description: "The organization role assigned to the identity. For custom roles, this is the role slug.",
				Computed:    true,
			},
			"custom_role_id": schema.StringAttribute{
				Description: "The ID of the custom organization role assigned to the identity. Null if the identity has a predefined role.",
				Computed:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The ID of the organization the identity belongs to",
				Computed:    true,
			},
			"metadata": schema.ListNestedAttribute{
				Description: "The metadata associated with the identity",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The key of the metadata entry",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the metadata entry",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *IdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentityDataSourceModel

	// Read Terraform configuration data into the model.
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgIdentity, err := d.client.GetIdentity(infisical.GetIdentityRequest{
		IdentityID: data.ID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Identity not found",
				fmt.Sprintf("No identity was found with ID %s", data.ID.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the identity",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	data.Name = types.StringValue(orgIdentity.Identity.Name)
	data.HasDeleteProtection = types.BoolValue(orgIdentity.Identity.HasDeleteProtection)
	data.OrgID = types.StringValue(orgIdentity.OrgID)

	authModes := make([]attr.Value, len(orgIdentity.Identity.AuthMethods))
	for i, method := range orgIdentity.Identity.AuthMethods {
		authModes[i] = types.StringValue(method)
	}
	data.AuthModes = types.ListValueMust(types.StringType, authModes)

	if orgIdentity.CustomRole != nil {
		data.Role = types.StringValue(orgIdentity.CustomRole.Slug)
		data.CustomRoleID = types.StringValue(orgIdentity.CustomRole.ID)
	} else {
		data.Role = types.StringValue(orgIdentity.Role)
		data.CustomRoleID = types.StringNull()
	}

	metadataObjectType := types.ObjectType{AttrTypes: identityMetadataAttrTypes}
	metadata := make([]IdentityDataSourceMetaEntry, 0, len(orgIdentity.Metadata))
	for _, m := range orgIdentity.Metadata {
		metadata = append(metadata, IdentityDataSourceMetaEntry{
			Key:   types.StringValue(m.Key),
			Value: types.StringValue(m.Value),
		})
	}
	metadataList, diags := types.ListValueFrom(ctx, metadataObjectType, metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Metadata = metadataList

	// Save data into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
