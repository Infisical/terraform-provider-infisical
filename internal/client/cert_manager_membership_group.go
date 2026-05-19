package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetCertManagerGroup    = "CallGetCertManagerGroup"
	operationAddCertManagerGroup    = "CallAddCertManagerGroup"
	operationUpdateCertManagerGroup = "CallUpdateCertManagerGroup"
	operationRemoveCertManagerGroup = "CallRemoveCertManagerGroup"
)

func (client Client) GetCertManagerGroup(request GetCertManagerGroupRequest) (GetCertManagerGroupResponse, error) {
	var groupResponse GetCertManagerGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/access/groups/%s", request.GroupId))

	if err != nil {
		return GetCertManagerGroupResponse{}, errors.NewGenericRequestError(operationGetCertManagerGroup, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertManagerGroupResponse{}, ErrNotFound
		}
		return GetCertManagerGroupResponse{}, errors.NewAPIErrorWithResponse(operationGetCertManagerGroup, response, nil)
	}

	return groupResponse, nil
}

func (client Client) AddCertManagerGroup(request AddCertManagerGroupRequest) (AddCertManagerGroupResponse, error) {
	var groupResponse AddCertManagerGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/access/groups/%s", request.GroupId))

	if err != nil {
		return AddCertManagerGroupResponse{}, errors.NewGenericRequestError(operationAddCertManagerGroup, err)
	}

	if response.IsError() {
		return AddCertManagerGroupResponse{}, errors.NewAPIErrorWithResponse(operationAddCertManagerGroup, response, nil)
	}

	return groupResponse, nil
}

func (client Client) UpdateCertManagerGroup(request UpdateCertManagerGroupRequest) (UpdateCertManagerGroupResponse, error) {
	var groupResponse UpdateCertManagerGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/access/groups/%s", request.GroupId))

	if err != nil {
		return UpdateCertManagerGroupResponse{}, errors.NewGenericRequestError(operationUpdateCertManagerGroup, err)
	}

	if response.IsError() {
		return UpdateCertManagerGroupResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertManagerGroup, response, nil)
	}

	return groupResponse, nil
}

func (client Client) RemoveCertManagerGroup(request RemoveCertManagerGroupRequest) (RemoveCertManagerGroupResponse, error) {
	var groupResponse RemoveCertManagerGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/access/groups/%s", request.GroupId))

	if err != nil {
		return RemoveCertManagerGroupResponse{}, errors.NewGenericRequestError(operationRemoveCertManagerGroup, err)
	}

	if response.IsError() {
		return RemoveCertManagerGroupResponse{}, errors.NewAPIErrorWithResponse(operationRemoveCertManagerGroup, response, nil)
	}

	return groupResponse, nil
}
