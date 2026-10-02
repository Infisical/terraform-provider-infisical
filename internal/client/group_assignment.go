package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

// These endpoints act on the organization the session is scoped to. Linking a root-org group
// into a sub-organization therefore requires a session scoped to that sub-organization
// (auth.organization_slug on the provider).

const (
	operationListAvailableGroups      = "CallListAvailableGroups"
	operationCreateOrgGroupMembership = "CallCreateOrgGroupMembership"
	operationGetOrgGroupMembership    = "CallGetOrgGroupMembership"
	operationListOrgGroupMemberships  = "CallListOrgGroupMemberships"
	operationUpdateOrgGroupMembership = "CallUpdateOrgGroupMembership"
	operationDeleteOrgGroupMembership = "CallDeleteOrgGroupMembership"
)

// ListAvailableGroups returns the parent-organization groups that can be linked into the
// sub-organization the session is scoped to.
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

// GetAvailableGroup finds a linkable parent-organization group by ID or, when groupID is empty,
// by slug. It returns ErrNotFound when the group is not in the available list.
func (client Client) GetAvailableGroup(groupID, groupSlug string) (AvailableGroup, error) {
	groups, err := client.ListAvailableGroups()
	if err != nil {
		return AvailableGroup{}, err
	}

	for _, group := range groups {
		if groupID != "" && group.ID == groupID {
			return group, nil
		}
		if groupID == "" && groupSlug != "" && group.Slug == groupSlug {
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

// GetOrgGroupMembership returns the membership of a group in the session's organization. It
// returns ErrNotFound when the group is not linked.
func (client Client) GetOrgGroupMembership(groupID string) (OrgGroupMembership, error) {
	if groupID == "" {
		return OrgGroupMembership{}, ErrNotFound
	}

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

// ListOrgGroupMemberships returns every group membership of the session's organization,
// paginating over the offset until the full set is retrieved.
func (client Client) ListOrgGroupMemberships() ([]OrgGroupMembership, error) {
	const pageSize = 100
	offset := 0
	var all []OrgGroupMembership

	for {
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
		offset += pageSize

		if len(responseData.GroupMemberships) == 0 || offset >= responseData.TotalCount {
			break
		}
	}

	return all, nil
}

// GetOrgGroupMembershipBySlug finds a linked group by its slug. It returns ErrNotFound when no
// linked group has that slug.
func (client Client) GetOrgGroupMembershipBySlug(groupSlug string) (OrgGroupMembership, error) {
	memberships, err := client.ListOrgGroupMemberships()
	if err != nil {
		return OrgGroupMembership{}, err
	}

	for _, membership := range memberships {
		if membership.Group.Slug == groupSlug {
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

// DeleteOrgGroupMembership unlinks a group from the session's organization. The API only
// allows this in a sub-organization the group is linked into, never in the group's own org.
func (client Client) DeleteOrgGroupMembership(groupID string) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/organizations/memberships/groups/%s", groupID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteOrgGroupMembership, err)
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteOrgGroupMembership, response, nil)
	}

	return nil
}
