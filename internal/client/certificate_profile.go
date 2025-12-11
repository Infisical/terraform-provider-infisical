package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateCertificateProfile = "CallCreateCertificateProfile"
	operationGetCertificateProfile    = "CallGetCertificateProfile"
	operationUpdateCertificateProfile = "CallUpdateCertificateProfile"
	operationDeleteCertificateProfile = "CallDeleteCertificateProfile"
)

func (client Client) CreateCertificateProfile(request CreateCertificateProfileRequest) (CreateCertificateProfileResponse, error) {
	var profileResponse CreateCertificateProfileResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profileResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/certificate-profiles")

	if err != nil {
		return CreateCertificateProfileResponse{}, errors.NewGenericRequestError(operationCreateCertificateProfile, err)
	}

	if response.IsError() {
		return CreateCertificateProfileResponse{}, errors.NewAPIErrorWithResponse(operationCreateCertificateProfile, response, nil)
	}

	return profileResponse, nil
}

func (client Client) GetCertificateProfile(request GetCertificateProfileRequest) (GetCertificateProfileResponse, error) {
	var profileResponse GetCertificateProfileResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profileResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificate-profiles/%s", request.ProfileId))

	if err != nil {
		return GetCertificateProfileResponse{}, errors.NewGenericRequestError(operationGetCertificateProfile, err)
	}

	if response.IsError() {
		return GetCertificateProfileResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificateProfile, response, nil)
	}

	return profileResponse, nil
}

func (client Client) UpdateCertificateProfile(request UpdateCertificateProfileRequest) (UpdateCertificateProfileResponse, error) {
	var profileResponse UpdateCertificateProfileResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profileResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/certificate-profiles/%s", request.ProfileId))

	if err != nil {
		return UpdateCertificateProfileResponse{}, errors.NewGenericRequestError(operationUpdateCertificateProfile, err)
	}

	if response.IsError() {
		return UpdateCertificateProfileResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertificateProfile, response, nil)
	}

	return profileResponse, nil
}

func (client Client) DeleteCertificateProfile(request DeleteCertificateProfileRequest) (DeleteCertificateProfileResponse, error) {
	var profileResponse DeleteCertificateProfileResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profileResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/certificate-profiles/%s", request.ProfileId))

	if err != nil {
		return DeleteCertificateProfileResponse{}, errors.NewGenericRequestError(operationDeleteCertificateProfile, err)
	}

	if response.IsError() {
		return DeleteCertificateProfileResponse{}, errors.NewAPIErrorWithResponse(operationDeleteCertificateProfile, response, nil)
	}

	return profileResponse, nil
}
