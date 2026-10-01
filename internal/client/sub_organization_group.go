package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

// These endpoints act on the organization the session is scoped to, so linking a root-organization
// group into a sub-organization requires a session scoped to that sub-organization
// (auth.organization_slug on the provider).

const (
	operationListAvailableGroups      = "CallListAvailableGroups"
	operationCreateOrgGroupMembership = "CallCreateOrgGroupMembership"
	operationGetOrgGroupMembership    = "CallGetOrgGroupMembership"
	operationListOrgGroupMemberships  = "CallListOrgGroupMemberships"
	operationUpdateOrgGroupMembership = "CallUpdateOrgGroupMembership"
	operationDeleteOrgGroupMembership = "CallDeleteOrgGroupMembership"
)

// ListAvailableGroups returns the root-organization groups that are not yet linked into the
// sub-organization the session is scoped to. It is always empty for a root-organization session.
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

// GetAvailableGroupBySlug returns ErrNotFound when no unlinked root-organization group has the slug.
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

// GetOrgGroupMembership returns ErrNotFound when the group is not linked to the session's organization.
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

// ListOrgGroupMemberships returns every group membership of the session's organization.
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

// GetOrgGroupMembershipBySlug returns ErrNotFound when no group linked to the session's
// organization has the slug.
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

// DeleteOrgGroupMembership unlinks a group from the session's organization. It returns
// ErrNotFound when the group is already gone or no longer linked.
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
