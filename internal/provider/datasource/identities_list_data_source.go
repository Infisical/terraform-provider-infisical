package datasource

import (
	"context"
	"fmt"
	"slices"

	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IdentitiesListDataSource{}

func NewIdentitiesListDataSource() datasource.DataSource {
	return &IdentitiesListDataSource{}
}

// IdentitiesListDataSource defines the data source implementation.
type IdentitiesListDataSource struct {
	client *infisical.Client
}

type IdentitiesListFilterModel struct {
	IdentityNames types.List `tfsdk:"identity_names"`
}

// IdentitiesListDataSourceModel describes the data source data model.
type IdentitiesListDataSourceModel struct {
	Scope      types.List                 `tfsdk:"scope"`
	Filter     *IdentitiesListFilterModel `tfsdk:"filter"`
	Identities types.List                 `tfsdk:"identities"`
}

// IdentitiesListIdentityModel describes a single identity in the list output.
type IdentitiesListIdentityModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	HasDeleteProtection types.Bool   `tfsdk:"has_delete_protection"`
	AuthModes           types.List   `tfsdk:"auth_modes"`
	Role                types.String `tfsdk:"role"`
	CustomRoleID        types.String `tfsdk:"custom_role_id"`
	OrgID               types.String `tfsdk:"org_id"`
	Scope               types.String `tfsdk:"scope"`
	ProjectID           types.String `tfsdk:"project_id"`
}

// identitiesListIdentityAttrTypes is the attribute type map for a single identity object.
var identitiesListIdentityAttrTypes = map[string]attr.Type{
	"id":                    types.StringType,
	"name":                  types.StringType,
	"has_delete_protection": types.BoolType,
	"auth_modes":            types.ListType{ElemType: types.StringType},
	"role":                  types.StringType,
	"custom_role_id":        types.StringType,
	"org_id":                types.StringType,
	"scope":                 types.StringType,
	"project_id":            types.StringType,
}

func (d *IdentitiesListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identities_list"
}

func (d *IdentitiesListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch the list of identities in the organization, from the organization scope, the project scope, or both. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			// Data source schemas have no Default field, so Read sets [organization, project]
			// when the user leaves scope unset.
			"scope": schema.ListAttribute{
				Description: "The scopes to list identities from. Each value must be `organization` or `project`. `organization` returns the identities that the organization owns. `project` returns the identities that a project owns, for the projects that the caller can access. Defaults to `[\"organization\", \"project\"]`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf(infisical.SearchIdentitiesScopeOrganization, infisical.SearchIdentitiesScopeProject),
					),
				},
			},
			"filter": schema.SingleNestedAttribute{
				Description: "The filters to apply to the identities list.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"identity_names": schema.ListAttribute{
						Description: "Only return identities with exactly these names. If the list is empty, no identities are returned. If not set, identities are not filtered by name.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"identities": schema.ListNestedAttribute{
				Description: "The identities in the requested scopes, sorted by name.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the identity",
							Computed:    true,
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
							Description: "The role assigned to the identity in its scope. For custom roles, this is the role slug. If the identity has more than one role, this is the first role.",
							Computed:    true,
						},
						"custom_role_id": schema.StringAttribute{
							Description: "The ID of the custom role assigned to the identity. Null if the identity has a predefined role.",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The ID of the organization the identity belongs to",
							Computed:    true,
						},
						"scope": schema.StringAttribute{
							Description: "The scope that owns the identity. Either `organization` or `project`.",
							Computed:    true,
						},
						"project_id": schema.StringAttribute{
							Description: "The ID of the project that owns the identity. Null for organization identities.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *IdentitiesListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentitiesListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to fetch identities",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentitiesListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	scope := []string{}
	if !data.Scope.IsNull() && !data.Scope.IsUnknown() {
		resp.Diagnostics.Append(data.Scope.ElementsAs(ctx, &scope, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		// defaults to both organization and project scope
		scope = []string{infisical.SearchIdentitiesScopeOrganization, infisical.SearchIdentitiesScopeProject}
	}

	scopeList, diags := types.ListValueFrom(ctx, types.StringType, scope)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Scope = scopeList

	memberships, err := d.client.ListIdentitiesV2(scope)
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching the identities",
			"If the error is not clear, please get in touch at infisical.com/slack\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
		return
	}

	var filterIdentityNames []string
	filterByName := false

	if data.Filter != nil {
		if !data.Filter.IdentityNames.IsNull() && !data.Filter.IdentityNames.IsUnknown() {
			filterByName = true
			resp.Diagnostics.Append(data.Filter.IdentityNames.ElementsAs(ctx, &filterIdentityNames, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	filteredMemberships := pkg.Filter(memberships, func(membership infisical.SearchIdentitiesV2Membership) bool {
		if filterByName {
			return slices.Contains(filterIdentityNames, membership.Identity.Name)
		}

		return true
	})

	identities := make([]IdentitiesListIdentityModel, 0, len(filteredMemberships))
	for _, membership := range filteredMemberships {

		authModes := membership.Identity.AuthMethods
		if authModes == nil {
			authModes = []string{}
		}
		authModeList, diags := types.ListValueFrom(ctx, types.StringType, authModes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		identity := IdentitiesListIdentityModel{
			ID:                  types.StringValue(membership.Identity.ID),
			Name:                types.StringValue(membership.Identity.Name),
			HasDeleteProtection: types.BoolValue(membership.Identity.HasDeleteProtection),
			AuthModes:           authModeList,
			Role:                types.StringNull(),
			CustomRoleID:        types.StringNull(),
			OrgID:               types.StringValue(membership.Identity.OrgID),
			Scope:               types.StringValue(membership.Scope),
			ProjectID:           types.StringPointerValue(membership.ProjectID),
		}

		if len(membership.Roles) > 0 {
			role := membership.Roles[0]
			if role.CustomRoleID != nil && *role.CustomRoleID != "" {
				identity.Role = types.StringPointerValue(role.CustomRoleSlug)
				identity.CustomRoleID = types.StringValue(*role.CustomRoleID)
			} else {
				identity.Role = types.StringValue(role.Role)
			}
		}

		identities = append(identities, identity)
	}

	identityList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: identitiesListIdentityAttrTypes}, identities)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Identities = identityList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
