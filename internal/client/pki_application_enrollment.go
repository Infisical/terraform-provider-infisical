package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetPkiApplicationEnrollment       = "CallGetPkiApplicationEnrollment"
	operationSetPkiApplicationApiEnrollment    = "CallSetPkiApplicationApiEnrollment"
	operationClearPkiApplicationApiEnrollment  = "CallClearPkiApplicationApiEnrollment"
	operationSetPkiApplicationEstEnrollment    = "CallSetPkiApplicationEstEnrollment"
	operationClearPkiApplicationEstEnrollment  = "CallClearPkiApplicationEstEnrollment"
	operationSetPkiApplicationAcmeEnrollment   = "CallSetPkiApplicationAcmeEnrollment"
	operationClearPkiApplicationAcmeEnrollment = "CallClearPkiApplicationAcmeEnrollment"
	operationRevealPkiApplicationAcmeEabSecret = "CallRevealPkiApplicationAcmeEabSecret"
	operationSetPkiApplicationScepEnrollment   = "CallSetPkiApplicationScepEnrollment"
	operationClearPkiApplicationScepEnrollment = "CallClearPkiApplicationScepEnrollment"
)

func (client Client) GetPkiApplicationEnrollment(request GetPkiApplicationEnrollmentRequest) (GetPkiApplicationEnrollmentResponse, error) {
	var enrollmentResponse GetPkiApplicationEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment", request.ApplicationId, request.ProfileId))

	if err != nil {
		return GetPkiApplicationEnrollmentResponse{}, errors.NewGenericRequestError(operationGetPkiApplicationEnrollment, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetPkiApplicationEnrollmentResponse{}, ErrNotFound
		}
		return GetPkiApplicationEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationGetPkiApplicationEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) SetPkiApplicationApiEnrollment(request SetPkiApplicationApiEnrollmentRequest) (SetPkiApplicationApiEnrollmentResponse, error) {
	var enrollmentResponse SetPkiApplicationApiEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Put(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/api", request.ApplicationId, request.ProfileId))

	if err != nil {
		return SetPkiApplicationApiEnrollmentResponse{}, errors.NewGenericRequestError(operationSetPkiApplicationApiEnrollment, err)
	}

	if response.IsError() {
		return SetPkiApplicationApiEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationSetPkiApplicationApiEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) ClearPkiApplicationApiEnrollment(request ClearPkiApplicationApiEnrollmentRequest) (ClearPkiApplicationApiEnrollmentResponse, error) {
	var enrollmentResponse ClearPkiApplicationApiEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/api", request.ApplicationId, request.ProfileId))

	if err != nil {
		return ClearPkiApplicationApiEnrollmentResponse{}, errors.NewGenericRequestError(operationClearPkiApplicationApiEnrollment, err)
	}

	if response.IsError() {
		return ClearPkiApplicationApiEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationClearPkiApplicationApiEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) SetPkiApplicationEstEnrollment(request SetPkiApplicationEstEnrollmentRequest) (SetPkiApplicationEstEnrollmentResponse, error) {
	var enrollmentResponse SetPkiApplicationEstEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Put(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/est", request.ApplicationId, request.ProfileId))

	if err != nil {
		return SetPkiApplicationEstEnrollmentResponse{}, errors.NewGenericRequestError(operationSetPkiApplicationEstEnrollment, err)
	}

	if response.IsError() {
		return SetPkiApplicationEstEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationSetPkiApplicationEstEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) ClearPkiApplicationEstEnrollment(request ClearPkiApplicationEstEnrollmentRequest) (ClearPkiApplicationEstEnrollmentResponse, error) {
	var enrollmentResponse ClearPkiApplicationEstEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/est", request.ApplicationId, request.ProfileId))

	if err != nil {
		return ClearPkiApplicationEstEnrollmentResponse{}, errors.NewGenericRequestError(operationClearPkiApplicationEstEnrollment, err)
	}

	if response.IsError() {
		return ClearPkiApplicationEstEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationClearPkiApplicationEstEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) SetPkiApplicationAcmeEnrollment(request SetPkiApplicationAcmeEnrollmentRequest) (SetPkiApplicationAcmeEnrollmentResponse, error) {
	var enrollmentResponse SetPkiApplicationAcmeEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Put(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/acme", request.ApplicationId, request.ProfileId))

	if err != nil {
		return SetPkiApplicationAcmeEnrollmentResponse{}, errors.NewGenericRequestError(operationSetPkiApplicationAcmeEnrollment, err)
	}

	if response.IsError() {
		return SetPkiApplicationAcmeEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationSetPkiApplicationAcmeEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) ClearPkiApplicationAcmeEnrollment(request ClearPkiApplicationAcmeEnrollmentRequest) (ClearPkiApplicationAcmeEnrollmentResponse, error) {
	var enrollmentResponse ClearPkiApplicationAcmeEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/acme", request.ApplicationId, request.ProfileId))

	if err != nil {
		return ClearPkiApplicationAcmeEnrollmentResponse{}, errors.NewGenericRequestError(operationClearPkiApplicationAcmeEnrollment, err)
	}

	if response.IsError() {
		return ClearPkiApplicationAcmeEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationClearPkiApplicationAcmeEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) RevealPkiApplicationAcmeEabSecret(request RevealPkiApplicationAcmeEabSecretRequest) (RevealPkiApplicationAcmeEabSecretResponse, error) {
	var eabResponse RevealPkiApplicationAcmeEabSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&eabResponse).
		SetHeader("User-Agent", USER_AGENT).
		Post(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/acme/eab/reveal", request.ApplicationId, request.ProfileId))

	if err != nil {
		return RevealPkiApplicationAcmeEabSecretResponse{}, errors.NewGenericRequestError(operationRevealPkiApplicationAcmeEabSecret, err)
	}

	if response.IsError() {
		return RevealPkiApplicationAcmeEabSecretResponse{}, errors.NewAPIErrorWithResponse(operationRevealPkiApplicationAcmeEabSecret, response, nil)
	}

	return eabResponse, nil
}

func (client Client) SetPkiApplicationScepEnrollment(request SetPkiApplicationScepEnrollmentRequest) (SetPkiApplicationScepEnrollmentResponse, error) {
	var enrollmentResponse SetPkiApplicationScepEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Put(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/scep", request.ApplicationId, request.ProfileId))

	if err != nil {
		return SetPkiApplicationScepEnrollmentResponse{}, errors.NewGenericRequestError(operationSetPkiApplicationScepEnrollment, err)
	}

	if response.IsError() {
		return SetPkiApplicationScepEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationSetPkiApplicationScepEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}

func (client Client) ClearPkiApplicationScepEnrollment(request ClearPkiApplicationScepEnrollmentRequest) (ClearPkiApplicationScepEnrollmentResponse, error) {
	var enrollmentResponse ClearPkiApplicationScepEnrollmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&enrollmentResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s/enrollment/scep", request.ApplicationId, request.ProfileId))

	if err != nil {
		return ClearPkiApplicationScepEnrollmentResponse{}, errors.NewGenericRequestError(operationClearPkiApplicationScepEnrollment, err)
	}

	if response.IsError() {
		return ClearPkiApplicationScepEnrollmentResponse{}, errors.NewAPIErrorWithResponse(operationClearPkiApplicationScepEnrollment, response, nil)
	}

	return enrollmentResponse, nil
}
