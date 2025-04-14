package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) CreateProjectIdentity(request CreateProjectIdentityRequest) (CreateProjectIdentityResponse, error) {
	var responeData CreateProjectIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/identity-memberships/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return CreateProjectIdentityResponse{}, fmt.Errorf("CallCreateProjectIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectIdentityResponse{}, fmt.Errorf("CallCreateProjectIdentity: Unsuccessful response. [response=%s]", response)
	}

	return responeData, nil
}

func (client Client) DeleteProjectIdentity(request DeleteProjectIdentityRequest) (DeleteProjectIdentityResponse, error) {
	var responseData DeleteProjectIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v2/workspace/%s/identity-memberships/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return DeleteProjectIdentityResponse{}, fmt.Errorf("CallDeleteProjectIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectIdentityResponse{}, fmt.Errorf("CallDeleteProjectIdentity: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) UpdateProjectIdentity(request UpdateProjectIdentityRequest) (UpdateProjectIdentityResponse, error) {
	var responseData UpdateProjectIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v2/workspace/%s/identity-memberships/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return UpdateProjectIdentityResponse{}, fmt.Errorf("CallUpdateProjectIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectIdentityResponse{}, fmt.Errorf("CallUpdateProjectIdentity: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) GetProjectIdentityByID(request GetProjectIdentityByIDRequest) (GetProjectIdentityByIDResponse, error) {
	var responseData GetProjectIdentityByIDResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v2/workspace/%s/identity-memberships/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return GetProjectIdentityByIDResponse{}, fmt.Errorf("GetProjectIdentityByIDResponse: Unable to complete api request [err=%s]", err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectIdentityByIDResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetProjectIdentityByIDResponse{}, fmt.Errorf("GetProjectIdentityByIDResponse: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}

func (client Client) GetProjectIdentityByMembershipID(request GetProjectIdentityByMembershipIDRequest) (GetProjectIdentityByIDResponse, error) {
	var responseData GetProjectIdentityByIDResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v2/workspace/identity-memberships/%s", request.MembershipID))

	if err != nil {
		return GetProjectIdentityByIDResponse{}, fmt.Errorf("GetProjectIdentityByMembershipID: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetProjectIdentityByIDResponse{}, ErrNotFound
		}
		return GetProjectIdentityByIDResponse{}, fmt.Errorf("GetProjectIdentityByMembershipID: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}
