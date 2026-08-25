package datasource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource                     = &IdentityDataSource{}
	_ datasource.DataSourceWithConfigValidators = &IdentityDataSource{}
)

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
		Description: "Look up an Infisical identity by its ID or its name. Returns the identity's details including its name, organization role, auth methods, and metadata. Identity names are not unique, so a lookup by name fails when more than one identity matches. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the identity to look up. Exactly one of id or name must be set.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the identity to look up. Exactly one of id or name must be set. Identity names are not unique, so the lookup fails if more than one identity has this name; use id to select a specific one.",
				Optional:    true,
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

func (d *IdentityDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

// resolveIdentityIDByName turns an identity name into the ID of the one identity that
// bears it. Identity names are not unique, so anything other than exactly one match is
// an error
func resolveIdentityIDByName(client *infisical.Client, name string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	result, err := client.SearchIdentitiesByName(name)
	if err != nil {
		diags.AddError(
			"Something went wrong while searching for the identity",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return "", diags
	}

	switch {
	case len(result.Identities) == 0:
		diags.AddError(
			"Identity not found",
			fmt.Sprintf("No identity was found with name %q", name),
		)
		return "", diags

	case len(result.Identities) > 1:
		ids := make([]string, 0, len(result.Identities))
		for _, match := range result.Identities {
			ids = append(ids, match.Identity.ID)
		}

		detail := fmt.Sprintf(
			"%d identities are named %q, so the name does not identify a single identity. "+
				"Set id to one of the following instead of name:\n  %s",
			result.TotalCount, name, strings.Join(ids, "\n  "),
		)
		// The search returns one page, so a pathological number of duplicates would be
		// truncated. Say so rather than presenting a partial list as the whole set.
		if result.TotalCount > len(ids) {
			detail += fmt.Sprintf("\n  ...and %d more", result.TotalCount-len(ids))
		}

		diags.AddError("Multiple identities match that name", detail)
		return "", diags
	}

	return result.Identities[0].Identity.ID, diags
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

	// ConfigValidators guarantees exactly one of id and name is set.
	identityID := data.ID.ValueString()
	if data.ID.IsNull() {
		resolvedID, diags := resolveIdentityIDByName(d.client, data.Name.ValueString())
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		identityID = resolvedID
	}

	// Both lookup paths finish with a fetch by ID. The search endpoint omits identity
	// metadata and auth
	orgIdentity, err := d.client.GetIdentity(infisical.GetIdentityRequest{
		IdentityID: identityID,
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Identity not found",
				fmt.Sprintf("No identity was found with ID %s", identityID),
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

	data.ID = types.StringValue(orgIdentity.Identity.ID)
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
