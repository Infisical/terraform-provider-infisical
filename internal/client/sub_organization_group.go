package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

// These endpoints act on whatever org the session is scoped to, so linking into a sub-org needs
// auth.organization_slug pointing at it.

const (
	operationListAvailableGroups      = "CallListAvailableGroups"
	operationCreateOrgGroupMembership = "CallCreateOrgGroupMembership"
	operationGetOrgGroupMembership    = "CallGetOrgGroupMembership"
	operationListOrgGroupMemberships  = "CallListOrgGroupMemberships"
	operationUpdateOrgGroupMembership = "CallUpdateOrgGroupMembership"
	operationDeleteOrgGroupMembership = "CallDeleteOrgGroupMembership"
)

// The org from auth.organization_slug, or the identity's own org when it's not set.
func (client Client) GetSessionOrganizationID() (string, error) {
	if client.Config.OrganizationSlug == "" {
		details, err := client.GetIdentityDetails()
		if err != nil {
			return "", err
		}
		return details.IdentityDetails.Organization.ID, nil
	}

	organization, err := client.GetOrganizationBySlug(client.Config.OrganizationSlug)
	if err != nil {
		return "", err
	}
	return organization.ID, nil
}

// Root groups not linked to the current sub-org yet. Always empty when scoped to the root org.
func (client Client) ListAvailableGroups() ([]AvailableGroup, error) {
	var responseData ListAvailableGroupsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/organization/available-groups")

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListAvailableGroups, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationListAvailableGroups, response, nil)
	}

	return responseData.Groups, nil
}

func (client Client) GetAvailableGroupBySlug(slug string) (AvailableGroup, error) {
	groups, err := client.ListAvailableGroups()
	if err != nil {
		return AvailableGroup{}, err
	}

	for _, group := range groups {
		if group.Slug == slug {
			return group, nil
		}
	}

	return AvailableGroup{}, ErrNotFound
}

func (client Client) CreateOrgGroupMembership(request CreateOrgGroupMembershipRequest) (OrgGroupMembership, error) {
	var responseData OrgGroupMembershipResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/organizations/memberships/groups/%s", request.GroupID))

	if err != nil {
		return OrgGroupMembership{}, errors.NewGenericRequestError(operationCreateOrgGroupMembership, err)
	}

	if response.IsError() {
		return OrgGroupMembership{}, errors.NewAPIErrorWithResponse(operationCreateOrgGroupMembership, response, nil)
	}

	return responseData.GroupMembership, nil
}

func (client Client) GetOrgGroupMembership(groupID string) (OrgGroupMembership, error) {
	var responseData OrgGroupMembershipResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/organizations/memberships/groups/%s", groupID))

	if err != nil {
		return OrgGroupMembership{}, errors.NewGenericRequestError(operationGetOrgGroupMembership, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return OrgGroupMembership{}, ErrNotFound
	}

	if response.IsError() {
		return OrgGroupMembership{}, errors.NewAPIErrorWithResponse(operationGetOrgGroupMembership, response, nil)
	}

	return responseData.GroupMembership, nil
}

func (client Client) ListOrgGroupMemberships() ([]OrgGroupMembership, error) {
	const pageSize = 100
	var all []OrgGroupMembership

	for offset := 0; ; offset += pageSize {
		var responseData ListOrgGroupMembershipsResponse
		response, err := client.Config.HttpClient.
			R().
			SetResult(&responseData).
			SetHeader("User-Agent", USER_AGENT).
			SetQueryParams(map[string]string{
				"limit":  fmt.Sprintf("%d", pageSize),
				"offset": fmt.Sprintf("%d", offset),
			}).
			Get("api/v1/organizations/memberships/groups")

		if err != nil {
			return nil, errors.NewGenericRequestError(operationListOrgGroupMemberships, err)
		}

		if response.IsError() {
			return nil, errors.NewAPIErrorWithResponse(operationListOrgGroupMemberships, response, nil)
		}

		all = append(all, responseData.GroupMemberships...)

		if len(responseData.GroupMemberships) == 0 || offset+pageSize >= responseData.TotalCount {
			return all, nil
		}
	}
}

func (client Client) GetOrgGroupMembershipBySlug(slug string) (OrgGroupMembership, error) {
	memberships, err := client.ListOrgGroupMemberships()
	if err != nil {
		return OrgGroupMembership{}, err
	}

	for _, membership := range memberships {
		if membership.Group.Slug == slug {
			return membership, nil
		}
	}

	return OrgGroupMembership{}, ErrNotFound
}

func (client Client) UpdateOrgGroupMembership(request UpdateOrgGroupMembershipRequest) (OrgGroupMembership, error) {
	var responseData OrgGroupMembershipResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/organizations/memberships/groups/%s", request.GroupID))

	if err != nil {
		return OrgGroupMembership{}, errors.NewGenericRequestError(operationUpdateOrgGroupMembership, err)
	}

	if response.IsError() {
		return OrgGroupMembership{}, errors.NewAPIErrorWithResponse(operationUpdateOrgGroupMembership, response, nil)
	}

	return responseData.GroupMembership, nil
}

// A 404 means the group itself is gone. A group that exists but isn't linked comes back as a 400.
func (client Client) DeleteOrgGroupMembership(groupID string) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/organizations/memberships/groups/%s", groupID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteOrgGroupMembership, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteOrgGroupMembership, response, nil)
	}

	return nil
}
