package infisicalclient

import "fmt"

func (client Client) CreatePermanentProjectIdentitySpecificPrivilege(request CreatePermanentProjectIdentitySpecificPrivilegeRequest) (CreateProjectIdentitySpecificPrivilegeResponse, error) {
	var responeData CreateProjectIdentitySpecificPrivilegeResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("/api/v1/additional-privilege/identity/permanent")

	if err != nil {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("CreatePermanentProjectIdentitySpecificPrivilege: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("CreatePermanentProjectIdentitySpecificPrivilege: Unsuccessful response. [response=%s]", response)
	}

	return responeData, nil
}

func (client Client) CreateTemporaryProjectIdentitySpecificPrivilege(request CreateTemporaryProjectIdentitySpecificPrivilegeRequest) (CreateProjectIdentitySpecificPrivilegeResponse, error) {
	var responeData CreateProjectIdentitySpecificPrivilegeResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("/api/v1/additional-privilege/identity/temporary")

	if err != nil {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("CreateTemporaryProjectIdentitySpecificPrivilege: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("CreateTemporaryProjectIdentitySpecificPrivilege: Unsuccessful response. [response=%s]", response)
	}

	return responeData, nil
}

func (client Client) CreateProjectIdentitySpecificPrivilegeV2(request CreateProjectIdentitySpecificPrivilegeV2Request) (CreateProjectIdentitySpecificPrivilegeV2Response, error) {
	var responeData CreateProjectIdentitySpecificPrivilegeV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("/api/v2/identity-project-additional-privilege")

	if err != nil {
		return CreateProjectIdentitySpecificPrivilegeV2Response{}, fmt.Errorf("CreateProjectIdentitySpecificPrivilegeV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectIdentitySpecificPrivilegeV2Response{}, fmt.Errorf("CreateProjectIdentitySpecificPrivilegeV2: Unsuccessful response. [response=%s]", response)
	}

	return responeData, nil
}

func (client Client) DeleteProjectIdentitySpecificPrivilege(request DeleteProjectIdentitySpecificPrivilegeRequest) (DeleteProjectIdentitySpecificPrivilegeResponse, error) {
	var responseData DeleteProjectIdentitySpecificPrivilegeResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("/api/v1/additional-privilege/identity")

	if err != nil {
		return DeleteProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("DeleteProjectIdentitySpecificPrivilege: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("DeleteProjectIdentitySpecificPrivilege: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) UpdateProjectIdentitySpecificPrivilege(request UpdateProjectIdentitySpecificPrivilegeRequest) (UpdateProjectIdentitySpecificPrivilegeResponse, error) {
	var responseData UpdateProjectIdentitySpecificPrivilegeResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("/api/v1/additional-privilege/identity")

	if err != nil {
		return UpdateProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("UpdateProjectIdentitySpecificPrivilege: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("UpdateProjectIdentitySpecificPrivilege: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) UpdateProjectIdentitySpecificPrivilegeV2(request UpdateProjectIdentitySpecificPrivilegeV2Request) (UpdateProjectIdentitySpecificPrivilegeV2Response, error) {
	var responseData UpdateProjectIdentitySpecificPrivilegeV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("/api/v2/identity-project-additional-privilege/%s", request.ID))

	if err != nil {
		return UpdateProjectIdentitySpecificPrivilegeV2Response{}, fmt.Errorf("UpdateProjectIdentitySpecificPrivilegeV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectIdentitySpecificPrivilegeV2Response{}, fmt.Errorf("UpdateProjectIdentitySpecificPrivilegeV2: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) GetProjectIdentitySpecificPrivilegeBySlug(request GetProjectIdentitySpecificPrivilegeRequest) (GetProjectIdentitySpecificPrivilegeResponse, error) {
	var responseData GetProjectIdentitySpecificPrivilegeResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("/api/v1/additional-privilege/identity/%s?projectSlug=%s&identityId=%s", request.PrivilegeSlug, request.ProjectSlug, request.IdentityID))

	if err != nil {
		return GetProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("GetProjectIdentitySpecificPrivilegeBySlug: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectIdentitySpecificPrivilegeResponse{}, fmt.Errorf("GetProjectIdentitySpecificPrivilegeBySlug: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) GetProjectIdentitySpecificPrivilegeV2(request GetProjectIdentitySpecificPrivilegeV2Request) (GetProjectIdentitySpecificPrivilegeV2Response, error) {
	var responseData GetProjectIdentitySpecificPrivilegeV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("/api/v2/identity-project-additional-privilege/%s", request.ID))

	if err != nil {
		return GetProjectIdentitySpecificPrivilegeV2Response{}, fmt.Errorf("GetProjectIdentitySpecificPrivilegeV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectIdentitySpecificPrivilegeV2Response{}, fmt.Errorf("GetProjectIdentitySpecificPrivilegeV2: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}
