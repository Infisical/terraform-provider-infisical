package datasource

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

var _ datasource.DataSource = &IdentitiesSearchDataSource{}

func NewIdentitiesSearchDataSource() datasource.DataSource {
	return &IdentitiesSearchDataSource{}
}

type IdentitiesSearchDataSource struct {
	client *infisical.Client
}

type IdentitiesSearchDataSourceModel struct {
	IdentityName types.String `tfsdk:"identity_name"`
	Mode         types.String `tfsdk:"mode"`
	Scope        types.String `tfsdk:"scope"`

	Identities   types.List  `tfsdk:"identities"`
	TotalCount  types.Int64 `tfsdk:"total_count"`
}

func (d *IdentitiesSearchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identities_search"
}

func (d *IdentitiesSearchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Search Infisical machine identities by name and return matching identity match objects.",
		Attributes: map[string]schema.Attribute{
			"identity_name": schema.StringAttribute{
				Description: "Identity name to search for.",
				Required:    true,
			},
			"mode": schema.StringAttribute{
				Description: "Name matching mode. Supported values: eq, contains.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"eq", "contains"}...),
				},
			},
			"scope": schema.StringAttribute{
				Description: "Search scope. Supported values: organization, project, both.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"organization", "project", "both"}...),
				},
			},
			"identities": schema.ListNestedAttribute{
				Description: "Matching identity match objects.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Match ID.",
							Computed:    true,
						},
						"identity_id": schema.StringAttribute{
							Description: "The identity ID.",
							Computed:    true,
						},
						"scope": schema.StringAttribute{
							Description: "Scope of the identity membership (organization or project).",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "Organization ID.",
							Computed:    true,
						},
						"project_id": schema.StringAttribute{
							Description: "Project ID (nullable).",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Last update timestamp.",
							Computed:    true,
						},
						"last_login_auth_method": schema.StringAttribute{
							Description: "Last login auth method (nullable).",
							Computed:    true,
						},
						"last_login_time": schema.StringAttribute{
							Description: "Last login time (nullable).",
							Computed:    true,
						},
						"project": schema.SingleNestedAttribute{
							Description: "Project details (nullable).",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{Computed: true},
								"name": schema.StringAttribute{Computed: true},
								"slug": schema.StringAttribute{Computed: true},
								"type": schema.StringAttribute{Computed: true},
							},
						},
						"roles": schema.ListNestedAttribute{
							Description: "Roles assigned to this identity (organization or project scope).",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{Computed: true},
									"role": schema.StringAttribute{Computed: true},
									"custom_role_id": schema.StringAttribute{Computed: true},
									"custom_role_name": schema.StringAttribute{Computed: true},
									"custom_role_slug": schema.StringAttribute{Computed: true},
									"custom_role_description": schema.StringAttribute{Computed: true},
									"is_temporary": schema.BoolAttribute{Computed: true},
									"temporary_mode": schema.StringAttribute{Computed: true},
									"temporary_range": schema.StringAttribute{Computed: true},
									"temporary_access_start_time": schema.StringAttribute{Computed: true},
									"temporary_access_end_time": schema.StringAttribute{Computed: true},
								},
							},
						},
						"identity": schema.SingleNestedAttribute{
							Description: "Identity details.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{Computed: true},
								"name": schema.StringAttribute{Computed: true},
								"has_delete_protection": schema.BoolAttribute{Computed: true},
								"org_id": schema.StringAttribute{Computed: true},
								"auth_methods": schema.ListAttribute{
									ElementType: types.StringType,
									Computed:    true,
								},
								"active_lockout_auth_methods": schema.ListAttribute{
									ElementType: types.StringType,
									Computed:    true,
								},
							},
						},
					},
				},
			},
			"total_count": schema.Int64Attribute{
				Description: "Total identities matching the filter (as reported by Infisical).",
				Computed:    true,
			},
		},
	}
}

func (d *IdentitiesSearchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentitiesSearchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if !d.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to search identities",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var data IdentitiesSearchDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mode := data.Mode.ValueString()
	scope := data.Scope.ValueString()

	// Data sources do not support schema defaults in this provider codebase.
	// When the user omits optional+computed fields, ValueString() can be empty.
	if mode == "" {
		mode = "contains"
	}
	if scope == "" {
		scope = "both"
	}

	var scopes []string
	switch scope {
	case "organization":
		scopes = []string{"organization"}
	case "project":
		scopes = []string{"project"}
	case "both":
		scopes = []string{"organization", "project"}
	default:
		resp.Diagnostics.AddError("Invalid scope", fmt.Sprintf("Unexpected scope value: %q", scope))
		return
	}

	identities, totalCount, err := d.client.SearchIdentitiesByName(infisical.SearchIdentityIDsByNameRequest{
		IdentityName: data.IdentityName.ValueString(),
		Mode:         mode,
		Scopes:       scopes,
		Limit:        100,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to search identities", "Error while searching identities: "+err.Error())
		return
	}

	// Prepare nested object types for `identities`.
	roleObjectAttrTypes := map[string]attr.Type{
		"id":                         types.StringType,
		"role":                        types.StringType,
		"custom_role_id":             types.StringType,
		"custom_role_name":           types.StringType,
		"custom_role_slug":           types.StringType,
		"custom_role_description":    types.StringType,
		"is_temporary":               types.BoolType,
		"temporary_mode":            types.StringType,
		"temporary_range":           types.StringType,
		"temporary_access_start_time": types.StringType,
		"temporary_access_end_time":   types.StringType,
	}
	roleObjectType := types.ObjectType{AttrTypes: roleObjectAttrTypes}

	projectObjectAttrTypes := map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"slug": types.StringType,
		"type": types.StringType,
	}
	projectObjectType := types.ObjectType{AttrTypes: projectObjectAttrTypes}

	identityObjectAttrTypes := map[string]attr.Type{
		"id":                         types.StringType,
		"name":                       types.StringType,
		"has_delete_protection":     types.BoolType,
		"org_id":                     types.StringType,
		"auth_methods":               types.ListType{ElemType: types.StringType},
		"active_lockout_auth_methods": types.ListType{ElemType: types.StringType},
	}
	identityObjectType := types.ObjectType{AttrTypes: identityObjectAttrTypes}

	matchObjectAttrTypes := map[string]attr.Type{
		"id":                         types.StringType,
		"identity_id":               types.StringType,
		"scope":                     types.StringType,
		"org_id":                    types.StringType,
		"project_id":                types.StringType,
		"created_at":                types.StringType,
		"updated_at":                types.StringType,
		"last_login_auth_method":   types.StringType,
		"last_login_time":           types.StringType,
		"project":                   projectObjectType,
		"roles":                     types.ListType{ElemType: roleObjectType},
		"identity":                  identityObjectType,
	}
	matchObjectType := types.ObjectType{AttrTypes: matchObjectAttrTypes}

	matchObjValues := make([]attr.Value, 0, len(identities))
	for _, m := range identities {
		// roles list
		roleObjValues := make([]attr.Value, 0, len(m.Roles))
		for _, r := range m.Roles {
			roleVals := map[string]attr.Value{
				"id":    types.StringValue(r.ID),
				"role":  types.StringValue(r.Role),
				"custom_role_id":          nullableStringToTF(r.CustomRoleID),
				"custom_role_name":        nullableStringToTF(r.CustomRoleName),
				"custom_role_slug":        nullableStringToTF(r.CustomRoleSlug),
				"custom_role_description": nullableStringToTF(r.CustomRoleDescription),
				"is_temporary":            types.BoolValue(r.IsTemporary),
				"temporary_mode":          nullableStringToTF(r.TemporaryMode),
				"temporary_range":         nullableStringToTF(r.TemporaryRange),
				"temporary_access_start_time": nullableStringToTF(r.TemporaryAccessStartTime),
				"temporary_access_end_time":   nullableStringToTF(r.TemporaryAccessEndTime),
			}
			roleObj, roleDiags := types.ObjectValue(roleObjectAttrTypes, roleVals)
			resp.Diagnostics.Append(roleDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			roleObjValues = append(roleObjValues, roleObj)
		}

		rolesList := types.ListValueMust(types.ObjectType{AttrTypes: roleObjectAttrTypes}, roleObjValues)

		// identity object
		authVals := make([]attr.Value, 0, len(m.Identity.AuthMethods))
		for _, am := range m.Identity.AuthMethods {
			authVals = append(authVals, types.StringValue(am))
		}
		if len(authVals) == 0 {
			authVals = []attr.Value{}
		}
		authList := types.ListValueMust(types.StringType, authVals)

		activeLockVals := make([]attr.Value, 0, len(m.Identity.ActiveLockoutAuthMethods))
		for _, am := range m.Identity.ActiveLockoutAuthMethods {
			activeLockVals = append(activeLockVals, types.StringValue(am))
		}
		activeLockList := types.ListValueMust(types.StringType, activeLockVals)

		identityVals := map[string]attr.Value{
			"id":                       types.StringValue(m.Identity.ID),
			"name":                     types.StringValue(m.Identity.Name),
			"has_delete_protection":   types.BoolValue(m.Identity.HasDeleteProtection),
			"org_id":                   types.StringValue(m.Identity.OrgID),
			"auth_methods":            authList,
			"active_lockout_auth_methods": activeLockList,
		}
		identityObj, identityDiags := types.ObjectValue(identityObjectAttrTypes, identityVals)
		resp.Diagnostics.Append(identityDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// project object
		var projectObj types.Object
		if m.Project != nil {
			projectVals := map[string]attr.Value{
				"id":   types.StringValue(m.Project.ID),
				"name": types.StringValue(m.Project.Name),
				"slug": types.StringValue(m.Project.Slug),
				"type": types.StringValue(m.Project.Type),
			}
			projectObjValue, projectDiags := types.ObjectValue(projectObjectAttrTypes, projectVals)
			resp.Diagnostics.Append(projectDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			projectObj = projectObjValue
		} else {
			projectObj = types.ObjectNull(projectObjectAttrTypes)
		}

		matchVals := map[string]attr.Value{
			"id":                       types.StringValue(m.ID),
			"identity_id":             types.StringValue(m.IdentityID),
			"scope":                    types.StringValue(m.Scope),
			"org_id":                   types.StringValue(m.OrgID),
			"project_id":               nullableStringToTF(m.ProjectID),
			"created_at":               types.StringValue(m.CreatedAt),
			"updated_at":               types.StringValue(m.UpdatedAt),
			"last_login_auth_method":  nullableStringToTF(m.LastLoginAuthMethod),
			"last_login_time":          nullableStringToTF(m.LastLoginTime),
			"project":                  projectObj,
			"roles":                    rolesList,
			"identity":                 identityObj,
		}

		matchObj, matchDiags := types.ObjectValue(matchObjectAttrTypes, matchVals)
		resp.Diagnostics.Append(matchDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		matchObjValues = append(matchObjValues, matchObj)
	}

	data.Identities = types.ListValueMust(matchObjectType, matchObjValues)
	data.TotalCount = types.Int64Value(int64(totalCount))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func nullableStringToTF(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}


