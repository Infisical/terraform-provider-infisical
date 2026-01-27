package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateOrgRole    = "CallCreateOrgRole"
	operationDeleteOrgRole    = "CallDeleteOrgRole"
	operationUpdateOrgRole    = "CallUpdateOrgRole"
	operationGetOrgRoleBySlug = "CallGetOrgRoleBySlug"
	operationGetOrgRoleById   = "CallGetOrgRoleById"
)

func (client Client) CreateOrgRole(request CreateOrgRoleRequest) (CreateOrgRoleResponse, error) {
	var responseData CreateOrgRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/organization/roles")

	if err != nil {
		return CreateOrgRoleResponse{}, errors.NewGenericRequestError(operationCreateOrgRole, err)
	}

	if response.IsError() {
		return CreateOrgRoleResponse{}, errors.NewAPIErrorWithResponse(operationCreateOrgRole, response, nil)
	}

	return responseData, nil
}

func (client Client) UpdateOrgRole(request UpdateOrgRoleRequest) (UpdateOrgRoleResponse, error) {
	var responseData UpdateOrgRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/organization/roles/%s", request.RoleId))

	if err != nil {
		return UpdateOrgRoleResponse{}, errors.NewGenericRequestError(operationUpdateOrgRole, err)
	}

	if response.IsError() {
		return UpdateOrgRoleResponse{}, errors.NewAPIErrorWithResponse(operationUpdateOrgRole, response, nil)
	}

	return responseData, nil
}

func (client Client) DeleteOrgRole(request DeleteOrgRoleRequest) (DeleteOrgRoleResponse, error) {
	var responseData DeleteOrgRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/organization/roles/%s", request.RoleId))

	if err != nil {
		return DeleteOrgRoleResponse{}, errors.NewGenericRequestError(operationDeleteOrgRole, err)
	}

	if response.IsError() {
		return DeleteOrgRoleResponse{}, errors.NewAPIErrorWithResponse(operationDeleteOrgRole, response, nil)
	}

	return responseData, nil
}

func (client Client) GetOrgRoleBySlug(request GetOrgRoleBySlugRequest) (GetOrgRoleBySlugResponse, error) {
	var responseData GetOrgRoleBySlugResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/organization/roles/slug/%s", request.RoleSlug))

	if err != nil {
		return GetOrgRoleBySlugResponse{}, errors.NewGenericRequestError(operationGetOrgRoleBySlug, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetOrgRoleBySlugResponse{}, ErrNotFound
		}
		return GetOrgRoleBySlugResponse{}, errors.NewAPIErrorWithResponse(operationGetOrgRoleBySlug, response, nil)
	}

	return responseData, nil
}

func (client Client) GetOrgRoleById(request GetOrgRoleByIdRequest) (GetOrgRoleByIdResponse, error) {
	var responseData GetOrgRoleByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/organization/roles/%s", request.RoleId))

	if err != nil {
		return GetOrgRoleByIdResponse{}, errors.NewGenericRequestError(operationGetOrgRoleById, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetOrgRoleByIdResponse{}, ErrNotFound
		}
		return GetOrgRoleByIdResponse{}, errors.NewAPIErrorWithResponse(operationGetOrgRoleById, response, nil)
	}

	return responseData, nil
}
