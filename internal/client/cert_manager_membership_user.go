package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationListCertManagerUsers   = "CallListCertManagerUsers"
	operationGetCertManagerUser     = "CallGetCertManagerUser"
	operationInviteCertManagerUsers = "CallInviteCertManagerUsers"
	operationUpdateCertManagerUser  = "CallUpdateCertManagerUser"
	operationRemoveCertManagerUser  = "CallRemoveCertManagerUser"
)

func (client Client) ListCertManagerUsers() (ListCertManagerUsersResponse, error) {
	var usersResponse ListCertManagerUsersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&usersResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/cert-manager/access/users")

	if err != nil {
		return ListCertManagerUsersResponse{}, errors.NewGenericRequestError(operationListCertManagerUsers, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return ListCertManagerUsersResponse{}, ErrNotFound
		}
		return ListCertManagerUsersResponse{}, errors.NewAPIErrorWithResponse(operationListCertManagerUsers, response, nil)
	}

	return usersResponse, nil
}

func (client Client) GetCertManagerUser(request GetCertManagerUserRequest) (GetCertManagerUserResponse, error) {
	var userResponse GetCertManagerUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&userResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/access/users/%s", request.UserId))

	if err != nil {
		return GetCertManagerUserResponse{}, errors.NewGenericRequestError(operationGetCertManagerUser, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertManagerUserResponse{}, ErrNotFound
		}
		return GetCertManagerUserResponse{}, errors.NewAPIErrorWithResponse(operationGetCertManagerUser, response, nil)
	}

	return userResponse, nil
}

func (client Client) InviteCertManagerUsers(request InviteCertManagerUsersRequest) (InviteCertManagerUsersResponse, error) {
	var usersResponse InviteCertManagerUsersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&usersResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/access/users")

	if err != nil {
		return InviteCertManagerUsersResponse{}, errors.NewGenericRequestError(operationInviteCertManagerUsers, err)
	}

	if response.IsError() {
		return InviteCertManagerUsersResponse{}, errors.NewAPIErrorWithResponse(operationInviteCertManagerUsers, response, nil)
	}

	return usersResponse, nil
}

func (client Client) UpdateCertManagerUser(request UpdateCertManagerUserRequest) (UpdateCertManagerUserResponse, error) {
	var userResponse UpdateCertManagerUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&userResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/access/users/%s", request.UserId))

	if err != nil {
		return UpdateCertManagerUserResponse{}, errors.NewGenericRequestError(operationUpdateCertManagerUser, err)
	}

	if response.IsError() {
		return UpdateCertManagerUserResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertManagerUser, response, nil)
	}

	return userResponse, nil
}

func (client Client) RemoveCertManagerUser(request RemoveCertManagerUserRequest) (RemoveCertManagerUserResponse, error) {
	var userResponse RemoveCertManagerUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&userResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/access/users/%s", request.UserId))

	if err != nil {
		return RemoveCertManagerUserResponse{}, errors.NewGenericRequestError(operationRemoveCertManagerUser, err)
	}

	if response.IsError() {
		return RemoveCertManagerUserResponse{}, errors.NewAPIErrorWithResponse(operationRemoveCertManagerUser, response, nil)
	}

	return userResponse, nil
}
