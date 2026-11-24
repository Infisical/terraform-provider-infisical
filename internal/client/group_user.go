package infisicalclient

import (
	"fmt"
	"net/http"
	"net/url"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationAddGroupUser    = "CallAddGroupUser"
	operationListGroupUsers  = "CallListGroupUsers"
	operationRemoveGroupUser = "CallRemoveGroupUser"
)

func (client Client) AddGroupUser(request AddGroupUserRequest) (GroupUser, error) {
	var responseData GroupUser
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Post(fmt.Sprintf("api/v1/groups/%s/users/%s", request.GroupID, url.PathEscape(request.Username)))

	if err != nil {
		return GroupUser{}, errors.NewGenericRequestError(operationAddGroupUser, err)
	}

	if response.IsError() {
		return GroupUser{}, errors.NewAPIErrorWithResponse(operationAddGroupUser, response, nil)
	}

	return responseData, nil
}

func (client Client) RemoveGroupUser(request RemoveGroupUserRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/groups/%s/users/%s", request.GroupID, url.PathEscape(request.Username)))

	if err != nil {
		return errors.NewGenericRequestError(operationRemoveGroupUser, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationRemoveGroupUser, response, nil)
	}

	return nil
}

func (client Client) ListGroupUsers(request ListGroupUsersRequest) ([]GroupUser, error) {
	const pageSize = 100
	const groupUsersFilterExistingMembers = "existingMembers"
	var all []GroupUser

	for offset := 0; ; offset += pageSize {
		var responseData ListGroupUsersResponse

		params := map[string]string{
			"limit":  fmt.Sprintf("%d", pageSize),
			"offset": fmt.Sprintf("%d", offset),
			"filter": groupUsersFilterExistingMembers,
		}
		if request.Username != "" {
			params["username"] = request.Username
		}

		response, err := client.Config.HttpClient.
			R().
			SetResult(&responseData).
			SetHeader("User-Agent", USER_AGENT).
			SetQueryParams(params).
			Get(fmt.Sprintf("api/v1/groups/%s/users", request.GroupID))

		if err != nil {
			return nil, errors.NewGenericRequestError(operationListGroupUsers, err)
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}

		if response.IsError() {
			return nil, errors.NewAPIErrorWithResponse(operationListGroupUsers, response, nil)
		}

		all = append(all, responseData.GroupUsers...)

		if len(responseData.GroupUsers) == 0 || offset+pageSize >= responseData.TotalCount {
			return all, nil
		}
	}
}
