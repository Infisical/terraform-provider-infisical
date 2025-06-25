package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectIdentity            = "CallCreateProjectIdentity"
	operationDeleteProjectIdentity            = "CallDeleteProjectIdentity"
	operationUpdateProjectIdentity            = "CallUpdateProjectIdentity"
	operationGetProjectIdentityByID           = "CallGetProjectIdentityByID"
	operationGetProjectIdentityByMembershipID = "CallGetProjectIdentityByMembershipID"
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
		return CreateProjectIdentityResponse{}, errors.NewGenericRequestError(operationCreateProjectIdentity, err)
	}

	if response.IsError() {
		return CreateProjectIdentityResponse{}, errors.NewAPIErrorWithResponse(operationCreateProjectIdentity, response, nil)
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
		return DeleteProjectIdentityResponse{}, errors.NewGenericRequestError(operationDeleteProjectIdentity, err)
	}

	if response.IsError() {
		return DeleteProjectIdentityResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectIdentity, response, nil)
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
		return UpdateProjectIdentityResponse{}, errors.NewGenericRequestError(operationUpdateProjectIdentity, err)
	}

	if response.IsError() {
		return UpdateProjectIdentityResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectIdentity, response, nil)
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
		return GetProjectIdentityByIDResponse{}, errors.NewGenericRequestError(operationGetProjectIdentityByID, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectIdentityByIDResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetProjectIdentityByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectIdentityByID, response, nil)
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
		return GetProjectIdentityByIDResponse{}, errors.NewGenericRequestError(operationGetProjectIdentityByMembershipID, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetProjectIdentityByIDResponse{}, ErrNotFound
		}
		return GetProjectIdentityByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectIdentityByMembershipID, response, nil)
	}

	return responseData, nil
}
