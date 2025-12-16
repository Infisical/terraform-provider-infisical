package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateInternalCA = "CallCreateInternalCA"
	operationGetCA            = "CallGetCA"
	operationGetInternalCA    = "CallGetInternalCA"
	operationGetACMECA        = "CallGetACMECA"
	operationGetADCSCA        = "CallGetADCSCA"
	operationUpdateInternalCA = "CallUpdateInternalCA"
	operationDeleteCA         = "CallDeleteCA"
	operationCreateACMECA     = "CallCreateACMECA"
	operationUpdateACMECA     = "CallUpdateACMECA"
	operationCreateADCSCA     = "CallCreateADCSCA"
	operationUpdateADCSCA     = "CallUpdateADCSCA"
)

func (client Client) CreateInternalCA(request CreateInternalCARequest) (CreateInternalCAResponse, error) {
	var caResponse CreateInternalCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/ca/internal")

	if err != nil {
		return CreateInternalCAResponse{}, errors.NewGenericRequestError(operationCreateInternalCA, err)
	}

	if response.IsError() {
		return CreateInternalCAResponse{}, errors.NewAPIErrorWithResponse(operationCreateInternalCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) GetCA(request GetCARequest) (GetCAResponse, error) {
	var caResponse GetCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/ca/%s", request.CAId))

	if err != nil {
		return GetCAResponse{}, errors.NewGenericRequestError(operationGetCA, err)
	}

	if response.IsError() {
		return GetCAResponse{}, errors.NewAPIErrorWithResponse(operationGetCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) GetInternalCA(request GetCARequest) (GetCAResponse, error) {
	var caResponse GetCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Get(fmt.Sprintf("api/v1/cert-manager/ca/internal/%s", request.CAId))

	if err != nil {
		return GetCAResponse{}, errors.NewGenericRequestError(operationGetInternalCA, err)
	}

	if response.IsError() {
		return GetCAResponse{}, errors.NewAPIErrorWithResponse(operationGetInternalCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) GetACMECA(request GetCARequest) (GetCAResponse, error) {
	var caResponse GetCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Get(fmt.Sprintf("api/v1/cert-manager/ca/acme/%s", request.CAId))

	if err != nil {
		return GetCAResponse{}, errors.NewGenericRequestError(operationGetACMECA, err)
	}

	if response.IsError() {
		return GetCAResponse{}, errors.NewAPIErrorWithResponse(operationGetACMECA, response, nil)
	}

	return caResponse, nil
}

func (client Client) GetADCSCA(request GetCARequest) (GetCAResponse, error) {
	var caResponse GetCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Get(fmt.Sprintf("api/v1/cert-manager/ca/azure-ad-cs/%s", request.CAId))

	if err != nil {
		return GetCAResponse{}, errors.NewGenericRequestError(operationGetADCSCA, err)
	}

	if response.IsError() {
		return GetCAResponse{}, errors.NewAPIErrorWithResponse(operationGetADCSCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) UpdateInternalCA(request UpdateInternalCARequest) (UpdateInternalCAResponse, error) {
	var caResponse UpdateInternalCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/ca/internal/%s", request.CAId))

	if err != nil {
		return UpdateInternalCAResponse{}, errors.NewGenericRequestError(operationUpdateInternalCA, err)
	}

	if response.IsError() {
		return UpdateInternalCAResponse{}, errors.NewAPIErrorWithResponse(operationUpdateInternalCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) DeleteInternalCA(request DeleteCARequest) (DeleteCAResponse, error) {
	var caResponse DeleteCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Delete(fmt.Sprintf("api/v1/cert-manager/ca/internal/%s", request.CAId))

	if err != nil {
		return DeleteCAResponse{}, errors.NewGenericRequestError(operationDeleteCA, err)
	}

	if response.IsError() {
		return DeleteCAResponse{}, errors.NewAPIErrorWithResponse(operationDeleteCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) DeleteACMECA(request DeleteCARequest) (DeleteCAResponse, error) {
	var caResponse DeleteCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Delete(fmt.Sprintf("api/v1/cert-manager/ca/acme/%s", request.CAId))

	if err != nil {
		return DeleteCAResponse{}, errors.NewGenericRequestError(operationDeleteCA, err)
	}

	if response.IsError() {
		return DeleteCAResponse{}, errors.NewAPIErrorWithResponse(operationDeleteCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) DeleteADCSCA(request DeleteCARequest) (DeleteCAResponse, error) {
	var caResponse DeleteCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Delete(fmt.Sprintf("api/v1/cert-manager/ca/azure-ad-cs/%s", request.CAId))

	if err != nil {
		return DeleteCAResponse{}, errors.NewGenericRequestError(operationDeleteCA, err)
	}

	if response.IsError() {
		return DeleteCAResponse{}, errors.NewAPIErrorWithResponse(operationDeleteCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) CreateACMECA(request CreateACMECARequest) (CreateACMECAResponse, error) {
	var caResponse CreateACMECAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/ca/acme")

	if err != nil {
		return CreateACMECAResponse{}, errors.NewGenericRequestError(operationCreateACMECA, err)
	}

	if response.IsError() {
		return CreateACMECAResponse{}, errors.NewAPIErrorWithResponse(operationCreateACMECA, response, nil)
	}

	return caResponse, nil
}

func (client Client) UpdateACMECA(request UpdateACMECARequest) (UpdateACMECAResponse, error) {
	var caResponse UpdateACMECAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/ca/acme/%s", request.CAId))

	if err != nil {
		return UpdateACMECAResponse{}, errors.NewGenericRequestError(operationUpdateACMECA, err)
	}

	if response.IsError() {
		return UpdateACMECAResponse{}, errors.NewAPIErrorWithResponse(operationUpdateACMECA, response, nil)
	}

	return caResponse, nil
}

func (client Client) CreateADCSCA(request CreateADCSCARequest) (CreateADCSCAResponse, error) {
	var caResponse CreateADCSCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/ca/azure-ad-cs")

	if err != nil {
		return CreateADCSCAResponse{}, errors.NewGenericRequestError(operationCreateADCSCA, err)
	}

	if response.IsError() {
		return CreateADCSCAResponse{}, errors.NewAPIErrorWithResponse(operationCreateADCSCA, response, nil)
	}

	return caResponse, nil
}

func (client Client) UpdateADCSCA(request UpdateADCSCARequest) (UpdateADCSCAResponse, error) {
	var caResponse UpdateADCSCAResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&caResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/ca/azure-ad-cs/%s", request.CAId))

	if err != nil {
		return UpdateADCSCAResponse{}, errors.NewGenericRequestError(operationUpdateADCSCA, err)
	}

	if response.IsError() {
		return UpdateADCSCAResponse{}, errors.NewAPIErrorWithResponse(operationUpdateADCSCA, response, nil)
	}

	return caResponse, nil
}
