package resource

import (
	"fmt"
	"sort"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CertManagerMemberRole struct {
	ID                      types.String `tfsdk:"id"`
	RoleSlug                types.String `tfsdk:"role_slug"`
	CustomRoleID            types.String `tfsdk:"custom_role_id"`
	IsTemporary             types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode           types.String `tfsdk:"temporary_mode"`
	TemporaryRange          types.String `tfsdk:"temporary_range"`
	TemporaryAccesStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime  types.String `tfsdk:"temporary_access_end_time"`
}

func certManagerRolesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Required:    true,
		Description: "The roles assigned to the cert manager membership",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Description: "The ID of the role assignment",
					Computed:    true,
				},
				"role_slug": schema.StringAttribute{
					Description: "The slug of the role",
					Required:    true,
				},
				"custom_role_id": schema.StringAttribute{
					Description: "The id of the custom role slug",
					Computed:    true,
					Optional:    true,
				},
				"is_temporary": schema.BoolAttribute{
					Description: "Flag to indicate the assigned role is temporary or not. When is_temporary is true fields temporary_mode, temporary_range and temporary_access_start_time is required.",
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
				},
				"temporary_mode": schema.StringAttribute{
					Description: "Type of temporary access given. Types: relative. Default: relative",
					Optional:    true,
					Computed:    true,
				},
				"temporary_range": schema.StringAttribute{
					Description: "TTL for the temporary time. Eg: 1m, 1h, 1d. Default: 1h",
					Optional:    true,
					Computed:    true,
				},
				"temporary_access_start_time": schema.StringAttribute{
					Description: "ISO time for which temporary access should begin. The current time is used by default.",
					Optional:    true,
					Computed:    true,
				},
				"temporary_access_end_time": schema.StringAttribute{
					Description: "ISO time for which temporary access will end. Computed based on temporary_range and temporary_access_start_time",
					Computed:    true,
					Optional:    true,
				},
			},
		},
	}
}

func certManagerBuildRoleUpdates(plan []CertManagerMemberRole, diags *diag.Diagnostics) ([]infisical.CertManagerMembershipRoleUpdate, bool, error) {
	var roles []infisical.CertManagerMembershipRoleUpdate
	var hasAtleastOnePermanentRole bool
	for _, el := range plan {
		isTemporary := el.IsTemporary.ValueBool()
		temporaryMode := el.TemporaryMode.ValueString()
		temporaryRange := el.TemporaryRange.ValueString()
		temporaryAccessStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
		}

		if isTemporary && (el.TemporaryAccesStartTime.IsNull() || el.TemporaryAccesStartTime.ValueString() == "") {
			diags.AddError(
				"Field temporary_access_start_time is required for temporary roles",
				fmt.Sprintf("Must provide valid ISO timestamp (YYYY-MM-DDTHH:MM:SSZ) for field temporary_access_start_time, role %s", el.RoleSlug.ValueString()),
			)
			return nil, false, nil
		}

		if el.TemporaryAccesStartTime.ValueString() != "" {
			parsed, err := time.Parse(time.RFC3339, el.TemporaryAccesStartTime.ValueString())
			if err != nil {
				return nil, false, fmt.Errorf("must provide valid ISO timestamp for field temporary_access_start_time %s, role %s", el.TemporaryAccesStartTime.ValueString(), el.RoleSlug.ValueString())
			}
			temporaryAccessStartTime = parsed
		}

		if isTemporary && temporaryMode == "" {
			temporaryMode = TEMPORARY_MODE_RELATIVE
		}
		if isTemporary && temporaryRange == "" {
			temporaryRange = TEMPORARY_RANGE_DEFAULT
		}

		roles = append(roles, infisical.CertManagerMembershipRoleUpdate{
			Role:                     el.RoleSlug.ValueString(),
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccessStartTime,
		})
	}
	return roles, hasAtleastOnePermanentRole, nil
}

func certManagerRolesFromAPI(apiRoles []infisical.CertManagerMembershipRole) []CertManagerMemberRole {
	result := make([]CertManagerMemberRole, 0, len(apiRoles))
	for _, el := range apiRoles {
		val := CertManagerMemberRole{
			ID:          types.StringValue(el.Id),
			RoleSlug:    types.StringValue(el.Role),
			IsTemporary: types.BoolValue(el.IsTemporary),
		}

		if el.CustomRoleId != nil {
			val.CustomRoleID = types.StringValue(*el.CustomRoleId)
			if el.CustomRoleSlug != nil {
				val.RoleSlug = types.StringValue(*el.CustomRoleSlug)
			}
		} else {
			val.CustomRoleID = types.StringValue("")
		}

		if el.IsTemporary {
			if el.TemporaryMode != nil {
				val.TemporaryMode = types.StringValue(*el.TemporaryMode)
			} else {
				val.TemporaryMode = types.StringNull()
			}
			if el.TemporaryRange != nil {
				val.TemporaryRange = types.StringValue(*el.TemporaryRange)
			} else {
				val.TemporaryRange = types.StringNull()
			}
			if el.TemporaryAccessStartTime != nil {
				val.TemporaryAccesStartTime = types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339))
			} else {
				val.TemporaryAccesStartTime = types.StringNull()
			}
			if el.TemporaryAccessEndTime != nil {
				val.TemporaryAccessEndTime = types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339))
			} else {
				val.TemporaryAccessEndTime = types.StringNull()
			}
		} else {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccesStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}

		result = append(result, val)
	}
	return result
}

func orderCertManagerRolesByPlan(planRoles []CertManagerMemberRole, apiRoles []CertManagerMemberRole) []CertManagerMemberRole {
	apiRoleMap := make(map[string]CertManagerMemberRole, len(apiRoles))
	for _, role := range apiRoles {
		apiRoleMap[role.RoleSlug.ValueString()] = role
	}

	ordered := make([]CertManagerMemberRole, 0, len(apiRoles))
	matched := make(map[string]bool)

	for _, planRole := range planRoles {
		slug := planRole.RoleSlug.ValueString()
		if apiRole, ok := apiRoleMap[slug]; ok && !matched[slug] {
			ordered = append(ordered, apiRole)
			matched[slug] = true
		}
	}

	var remaining []CertManagerMemberRole
	for _, apiRole := range apiRoles {
		if !matched[apiRole.RoleSlug.ValueString()] {
			remaining = append(remaining, apiRole)
		}
	}
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].RoleSlug.ValueString() < remaining[j].RoleSlug.ValueString()
	})
	ordered = append(ordered, remaining...)

	return ordered
}
