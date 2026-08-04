package infisicalclient

// GetOrganizationBySlug resolves an organization by its slug. Infisical exposes no
// lookup-by-slug endpoint that machine identities can call, so this checks the
// organization the caller is authenticated to first and then falls back to scanning
// the sub-organizations of the caller's root organization. It returns ErrNotFound
// when no organization visible to the caller carries the given slug.
func (client Client) GetOrganizationBySlug(slug string) (Organization, error) {
	identityDetails, err := client.GetIdentityDetails()
	if err != nil {
		return Organization{}, err
	}

	currentOrg := identityDetails.IdentityDetails.Organization
	if currentOrg.Slug == slug {
		return Organization{
			ID:   currentOrg.ID,
			Name: currentOrg.Name,
			Slug: currentOrg.Slug,
		}, nil
	}

	subOrgs, err := client.ListSubOrganizations()
	if err != nil {
		return Organization{}, err
	}

	for _, subOrg := range subOrgs {
		if subOrg.Slug == slug {
			return Organization{
				ID:   subOrg.ID,
				Name: subOrg.Name,
				Slug: subOrg.Slug,
			}, nil
		}
	}

	return Organization{}, ErrNotFound
}
