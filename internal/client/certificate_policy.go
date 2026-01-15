package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateCertificatePolicy = "CallCreateCertificatePolicy"
	operationGetCertificatePolicy    = "CallGetCertificatePolicy"
	operationUpdateCertificatePolicy = "CallUpdateCertificatePolicy"
	operationDeleteCertificatePolicy = "CallDeleteCertificatePolicy"
)

func (client Client) CreateCertificatePolicy(request CreateCertificatePolicyRequest) (CreateCertificatePolicyResponse, error) {
	var policyResponse CreateCertificatePolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&policyResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/certificate-policies")

	if err != nil {
		return CreateCertificatePolicyResponse{}, errors.NewGenericRequestError(operationCreateCertificatePolicy, err)
	}

	if response.IsError() {
		return CreateCertificatePolicyResponse{}, errors.NewAPIErrorWithResponse(operationCreateCertificatePolicy, response, nil)
	}

	return policyResponse, nil
}

func (client Client) GetCertificatePolicy(request GetCertificatePolicyRequest) (GetCertificatePolicyResponse, error) {
	var policyResponse GetCertificatePolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&policyResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificate-policies/%s", request.PolicyId))

	if err != nil {
		return GetCertificatePolicyResponse{}, errors.NewGenericRequestError(operationGetCertificatePolicy, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertificatePolicyResponse{}, ErrNotFound
		}
		return GetCertificatePolicyResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificatePolicy, response, nil)
	}

	return policyResponse, nil
}

func (client Client) UpdateCertificatePolicy(request UpdateCertificatePolicyRequest) (UpdateCertificatePolicyResponse, error) {
	var policyResponse UpdateCertificatePolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&policyResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/certificate-policies/%s", request.PolicyId))

	if err != nil {
		return UpdateCertificatePolicyResponse{}, errors.NewGenericRequestError(operationUpdateCertificatePolicy, err)
	}

	if response.IsError() {
		return UpdateCertificatePolicyResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertificatePolicy, response, nil)
	}

	return policyResponse, nil
}

func (client Client) DeleteCertificatePolicy(request DeleteCertificatePolicyRequest) (DeleteCertificatePolicyResponse, error) {
	var policyResponse DeleteCertificatePolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&policyResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/certificate-policies/%s", request.PolicyId))

	if err != nil {
		return DeleteCertificatePolicyResponse{}, errors.NewGenericRequestError(operationDeleteCertificatePolicy, err)
	}

	if response.IsError() {
		return DeleteCertificatePolicyResponse{}, errors.NewAPIErrorWithResponse(operationDeleteCertificatePolicy, response, nil)
	}

	return policyResponse, nil
}


