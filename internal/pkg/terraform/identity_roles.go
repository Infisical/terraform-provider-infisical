package terraform

import (
	"fmt"
	"sort"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const TemporaryModeRelative = "relative"

// IdentityRole describes a role assignment on a project identity.
type IdentityRole struct {
	ID                      types.String `tfsdk:"id"`
	RoleSlug                types.String `tfsdk:"role_slug"`
	CustomRoleID            types.String `tfsdk:"custom_role_id"`
	IsTemporary             types.Bool   `tfsdk:"is_temporary"`
	TemporaryMode           types.String `tfsdk:"temporary_mode"`
	TemporaryRange          types.String `tfsdk:"temporary_range"`
	TemporaryAccesStartTime types.String `tfsdk:"temporary_access_start_time"`
	TemporaryAccessEndTime  types.String `tfsdk:"temporary_access_end_time"`
}

// OrderAPIRolesByPlan reorders API-returned roles to match the ordering specified in the Terraform plan.
// The API may return roles in a non-deterministic order, but Terraform requires list elements to maintain
// the same positional order as the plan to avoid "inconsistent result after apply" errors.
func OrderAPIRolesByPlan(planRoles []IdentityRole, apiRoles []IdentityRole) []IdentityRole {
	apiRoleMap := make(map[string]IdentityRole, len(apiRoles))
	for _, role := range apiRoles {
		apiRoleMap[role.RoleSlug.ValueString()] = role
	}

	ordered := make([]IdentityRole, 0, len(apiRoles))
	matched := make(map[string]bool)

	for _, planRole := range planRoles {
		slug := planRole.RoleSlug.ValueString()
		if apiRole, ok := apiRoleMap[slug]; ok && !matched[slug] {
			ordered = append(ordered, apiRole)
			matched[slug] = true
		}
	}

	var remaining []IdentityRole
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

// BuildIdentityRequestRoles converts the roles declared in a Terraform plan into the
// API request representation. It applies the defaults used across the provider for temporary
// roles (relative mode, 1h range) and validates the role list: at least one permanent role is
// required, role slugs must be unique, temporary-only fields may not be set on permanent roles,
// and temporary_access_start_time must be in canonical RFC3339 UTC form.
func BuildIdentityRequestRoles(planRoles []IdentityRole) ([]infisical.CreateProjectIdentityRequestRoles, error) {
	var roles []infisical.CreateProjectIdentityRequestRoles
	var hasAtleastOnePermanentRole bool
	seenSlugs := make(map[string]bool, len(planRoles))
	for _, el := range planRoles {
		slug := el.RoleSlug.ValueString()
		if seenSlugs[slug] {
			return nil, fmt.Errorf("duplicate role_slug %q: each role may only appear once", slug)
		}
		seenSlugs[slug] = true

		isTemporary := el.IsTemporary.ValueBool()
		temporaryMode := el.TemporaryMode.ValueString()
		temporaryRange := el.TemporaryRange.ValueString()
		temporaryAccesStartTime := time.Now().UTC()

		if !isTemporary {
			hasAtleastOnePermanentRole = true
			if temporaryMode != "" || temporaryRange != "" || el.TemporaryAccesStartTime.ValueString() != "" {
				return nil, fmt.Errorf("role %q is permanent (is_temporary = false) but sets temporary_mode/temporary_range/temporary_access_start_time; set is_temporary = true or remove those fields", slug)
			}
		}

		if el.TemporaryAccesStartTime.ValueString() != "" {
			raw := el.TemporaryAccesStartTime.ValueString()
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return nil, fmt.Errorf("must provide valid ISO timestamp for field temporary_access_start_time %q, role %q", raw, slug)
			}
			if canonical := parsed.UTC().Format(time.RFC3339); canonical != raw {
				return nil, fmt.Errorf("temporary_access_start_time %q for role %q must be in canonical RFC3339 UTC format (e.g. %q)", raw, slug, canonical)
			}
			temporaryAccesStartTime = parsed
		}

		if isTemporary && temporaryMode == "" {
			temporaryMode = TemporaryModeRelative
		}
		if isTemporary && temporaryRange == "" {
			temporaryRange = "1h"
		}

		roles = append(roles, infisical.CreateProjectIdentityRequestRoles{
			Role:                     slug,
			IsTemporary:              isTemporary,
			TemporaryMode:            temporaryMode,
			TemporaryRange:           temporaryRange,
			TemporaryAccessStartTime: temporaryAccesStartTime,
		})
	}

	if !hasAtleastOnePermanentRole {
		return nil, fmt.Errorf("must have atleast one permanent role")
	}

	return roles, nil
}

// MapAPIRolesToIdentityModel converts API project member roles into the Terraform model
// representation, resolving custom role slugs and nulling temporary-only fields for permanent roles.
func MapAPIRolesToIdentityModel(apiRoles []infisical.ProjectMemberRole) []IdentityRole {
	result := make([]IdentityRole, 0, len(apiRoles))
	for _, el := range apiRoles {
		val := IdentityRole{
			ID:                      types.StringValue(el.ID),
			RoleSlug:                types.StringValue(el.Role),
			TemporaryAccessEndTime:  types.StringValue(el.TemporaryAccessEndTime.Format(time.RFC3339)),
			TemporaryRange:          types.StringValue(el.TemporaryRange),
			TemporaryMode:           types.StringValue(el.TemporaryMode),
			CustomRoleID:            types.StringValue(el.CustomRoleId),
			IsTemporary:             types.BoolValue(el.IsTemporary),
			TemporaryAccesStartTime: types.StringValue(el.TemporaryAccessStartTime.Format(time.RFC3339)),
		}
		if el.CustomRoleSlug != "" {
			val.RoleSlug = types.StringValue(el.CustomRoleSlug)
		}
		if !el.IsTemporary {
			val.TemporaryMode = types.StringNull()
			val.TemporaryRange = types.StringNull()
			val.TemporaryAccesStartTime = types.StringNull()
			val.TemporaryAccessEndTime = types.StringNull()
		}
		result = append(result, val)
	}
	return result
}
