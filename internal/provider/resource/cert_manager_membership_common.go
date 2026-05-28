package resource

import (
	infisical "terraform-provider-infisical/internal/client"
)

func firstRole(apiRoles []infisical.CertManagerMembershipRole, fallback string) string {
	if len(apiRoles) == 0 {
		return fallback
	}
	r := apiRoles[0]
	if r.CustomRoleSlug != nil {
		return *r.CustomRoleSlug
	}
	return r.Role
}
