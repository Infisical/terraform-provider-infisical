package infisicalclient

import "fmt"

// GetOrganizationBySlug resolves an organization by slug. It checks the identity's own org, then scans sub-orgs of its
// root. ErrNotFound is returned only when both lookups succeed and neither matches.
//
// Both sources always target the identity's root org, even when the session is scoped
// to a sub-org via auth.organization_slug. A Machine Identity created inside a sub-org has no root
// membership, so both sources fail; either failure is surfaced rather than treated as
// ErrNotFound, since an unavailable source cannot prove absence.
func (client Client) GetOrganizationBySlug(slug string) (Organization, error) {
	identityDetails, detailsErr := client.GetIdentityDetails()
	if detailsErr == nil {
		ownOrg := identityDetails.IdentityDetails.Organization
		if ownOrg.Slug == slug {
			// IdentityOrganization and Organization carry the same fields, so a conversion is enough
			return Organization(ownOrg), nil
		}
	}

	subOrgs, listErr := client.ListSubOrganizations()
	if listErr == nil {
		for _, subOrg := range subOrgs {
			if subOrg.Slug == slug {
				return Organization{
					ID:   subOrg.ID,
					Name: subOrg.Name,
					Slug: subOrg.Slug,
				}, nil
			}
		}
	}

	switch {
	case detailsErr != nil && listErr != nil:
		return Organization{}, fmt.Errorf(
			"neither organization lookup succeeded, so the slug %q could not be resolved. "+
				"This is expected when the machine identity was created inside a sub-organization: "+
				"organization lookups resolve against the root organization, which such an identity is not a member of. "+
				"Use a machine identity that belongs to the root organization, or supply the organization ID directly.\n"+
				"identity organization lookup: %w\nsub-organization lookup: %w",
			slug, detailsErr, listErr)
	case detailsErr != nil:
		return Organization{}, fmt.Errorf(
			"slug %q was not found in sub-organizations, and the identity's own organization lookup failed: %w",
			slug, detailsErr)
	case listErr != nil:
		return Organization{}, fmt.Errorf(
			"slug %q did not match the identity organization, and listing sub-organizations failed: %w",
			slug, listErr)
	}

	return Organization{}, ErrNotFound
}
