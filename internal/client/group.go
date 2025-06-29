package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) CreateGroup(request CreateGroupRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/groups")

	if err != nil {
		return Group{}, fmt.Errorf("CreateGroup: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return Group{}, fmt.Errorf("CreateGroup: Unsuccessful response. [response=%s]", response)
	}

	return groupResponse, nil
}

func (client Client) UpdateGroup(request UpdateGroupRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if err != nil {
		return Group{}, fmt.Errorf("UpdateGroup: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return Group{}, fmt.Errorf("UpdateGroup: Unsuccessful response. [response=%s]", response)
	}

	return groupResponse, nil
}

func (client Client) DeleteGroup(request DeleteGroupRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if err != nil {
		return Group{}, fmt.Errorf("DeleteGroup: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return Group{}, fmt.Errorf("DeleteGroup: Unsuccessful response. [response=%s]", response)
	}

	return groupResponse, nil
}

func (client Client) GetGroupById(request GetGroupByIdRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return Group{}, ErrNotFound
	}

	if err != nil {
		return Group{}, fmt.Errorf("CallGetGroupById: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return Group{}, fmt.Errorf("CallGetGroupById: Unsuccessful response. [response=%s]", response)
	}

	return groupResponse, nil
}

func (client Client) GetGroups() (GetGroupsResponse, error) {
	var body GetGroupsResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/groups")

	if err != nil {
		return GetGroupsResponse{}, fmt.Errorf("GetGroups: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetGroupsResponse{}, fmt.Errorf("GetGroups: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body, nil
}
