package infisicalclient

import "fmt"

func (client Client) CreateProjectRole(request CreateProjectRoleRequest) (CreateProjectRoleResponse, error) {
	var responeData CreateProjectRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/workspace/%s/roles", request.ProjectSlug))

	if err != nil {
		return CreateProjectRoleResponse{}, fmt.Errorf("CreateProjectRole: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectRoleResponse{}, fmt.Errorf("CreateProjectRole: Unsuccessful response. [response=%s]", response)
	}

	return responeData, nil
}

func (client Client) CreateProjectRoleV2(request CreateProjectRoleV2Request) (CreateProjectRoleV2Response, error) {
	var responseData CreateProjectRoleV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/roles", request.ProjectId))

	if err != nil {
		return CreateProjectRoleV2Response{}, fmt.Errorf("CreateProjectRoleV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectRoleV2Response{}, fmt.Errorf("CreateProjectRoleV2: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) DeleteProjectRole(request DeleteProjectRoleRequest) (DeleteProjectRoleResponse, error) {
	var responseData DeleteProjectRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v1/workspace/%s/roles/%s", request.ProjectSlug, request.RoleId))

	if err != nil {
		return DeleteProjectRoleResponse{}, fmt.Errorf("DeleteProjectRole: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectRoleResponse{}, fmt.Errorf("DeleteProjectRole: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) UpdateProjectRole(request UpdateProjectRoleRequest) (UpdateProjectRoleResponse, error) {
	var responseData UpdateProjectRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/workspace/%s/roles/%s", request.ProjectSlug, request.RoleId))

	if err != nil {
		return UpdateProjectRoleResponse{}, fmt.Errorf("UpdateProjectRole: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectRoleResponse{}, fmt.Errorf("UpdateProjectRole: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) UpdateProjectRoleV2(request UpdateProjectRoleV2Request) (UpdateProjectRoleV2Response, error) {
	var responseData UpdateProjectRoleV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v2/workspace/%s/roles/%s", request.ProjectId, request.RoleId))

	if err != nil {
		return UpdateProjectRoleV2Response{}, fmt.Errorf("UpdateProjectRoleV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectRoleV2Response{}, fmt.Errorf("UpdateProjectRoleV2: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) GetProjectRoleBySlug(request GetProjectRoleBySlugRequest) (GetProjectRoleBySlugResponse, error) {
	var responseData GetProjectRoleBySlugResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v1/workspace/%s/roles/slug/%s", request.ProjectSlug, request.RoleSlug))

	if err != nil {
		return GetProjectRoleBySlugResponse{}, fmt.Errorf("GetProjectRoleBySlug: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectRoleBySlugResponse{}, fmt.Errorf("GetProjectRoleBySlug: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) GetProjectRoleBySlugV2(request GetProjectRoleBySlugV2Request) (GetProjectRoleBySlugV2Response, error) {
	var responseData GetProjectRoleBySlugV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v2/workspace/%s/roles/slug/%s", request.ProjectId, request.RoleSlug))

	if err != nil {
		return GetProjectRoleBySlugV2Response{}, fmt.Errorf("GetProjectRoleBySlugV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectRoleBySlugV2Response{}, fmt.Errorf("GetProjectRoleBySlugV2: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}
