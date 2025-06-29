package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePermanentProjectIdentitySpecificPrivilege = "CallCreatePermanentProjectIdentitySpecificPrivilege"
	operationCreateTemporaryProjectIdentitySpecificPrivilege = "CallCreateTemporaryProjectIdentitySpecificPrivilege"
	operationCreateProjectIdentitySpecificPrivilegeV2        = "CallCreateProjectIdentitySpecificPrivilegeV2"
	operationDeleteProjectIdentitySpecificPrivilege          = "CallDeleteProjectIdentitySpecificPrivilege"
	operationUpdateProjectIdentitySpecificPrivilege          = "CallUpdateProjectIdentitySpecificPrivilege"
	operationUpdateProjectIdentitySpecificPrivilegeV2        = "CallUpdateProjectIdentitySpecificPrivilegeV2"
	operationGetProjectIdentitySpecificPrivilegeBySlug       = "CallGetProjectIdentitySpecificPrivilegeBySlug"
	operationGetProjectIdentitySpecificPrivilegeV2           = "CallGetProjectIdentitySpecificPrivilegeV2"
)

func (client Client) CreatePermanentProjectIdentitySpecificPrivilege(request CreatePermanentProjectIdentitySpecificPrivilegeRequest) (CreateProjectIdentitySpecificPrivilegeResponse, error) {
	var responeData CreateProjectIdentitySpecificPrivilegeResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("/api/v1/additional-privilege/identity/permanent")

	if err != nil {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, errors.NewGenericRequestError(operationCreatePermanentProjectIdentitySpecificPrivilege, err)
	}

	if response.IsError() {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, errors.NewAPIErrorWithResponse(operationCreatePermanentProjectIdentitySpecificPrivilege, response, nil)
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
		return CreateProjectIdentitySpecificPrivilegeResponse{}, errors.NewGenericRequestError(operationCreateTemporaryProjectIdentitySpecificPrivilege, err)
	}

	if response.IsError() {
		return CreateProjectIdentitySpecificPrivilegeResponse{}, errors.NewAPIErrorWithResponse(operationCreateTemporaryProjectIdentitySpecificPrivilege, response, nil)
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
		return CreateProjectIdentitySpecificPrivilegeV2Response{}, errors.NewGenericRequestError(operationCreateProjectIdentitySpecificPrivilegeV2, err)
	}

	if response.IsError() {
		return CreateProjectIdentitySpecificPrivilegeV2Response{}, errors.NewAPIErrorWithResponse(operationCreateProjectIdentitySpecificPrivilegeV2, response, nil)
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
		return DeleteProjectIdentitySpecificPrivilegeResponse{}, errors.NewGenericRequestError(operationDeleteProjectIdentitySpecificPrivilege, err)
	}

	if response.IsError() {
		return DeleteProjectIdentitySpecificPrivilegeResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectIdentitySpecificPrivilege, response, nil)
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
		return UpdateProjectIdentitySpecificPrivilegeResponse{}, errors.NewGenericRequestError(operationUpdateProjectIdentitySpecificPrivilege, err)
	}

	if response.IsError() {
		return UpdateProjectIdentitySpecificPrivilegeResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectIdentitySpecificPrivilege, response, nil)
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
		return UpdateProjectIdentitySpecificPrivilegeV2Response{}, errors.NewGenericRequestError(operationUpdateProjectIdentitySpecificPrivilegeV2, err)
	}

	if response.IsError() {
		return UpdateProjectIdentitySpecificPrivilegeV2Response{}, errors.NewAPIErrorWithResponse(operationUpdateProjectIdentitySpecificPrivilegeV2, response, nil)
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
		return GetProjectIdentitySpecificPrivilegeResponse{}, errors.NewGenericRequestError(operationGetProjectIdentitySpecificPrivilegeBySlug, err)
	}

	if response.IsError() {
		return GetProjectIdentitySpecificPrivilegeResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectIdentitySpecificPrivilegeBySlug, response, nil)
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
		return GetProjectIdentitySpecificPrivilegeV2Response{}, errors.NewGenericRequestError(operationGetProjectIdentitySpecificPrivilegeV2, err)
	}

	if response.IsError() {
		return GetProjectIdentitySpecificPrivilegeV2Response{}, errors.NewAPIErrorWithResponse(operationGetProjectIdentitySpecificPrivilegeV2, response, nil)
	}

	return responseData, nil
}
