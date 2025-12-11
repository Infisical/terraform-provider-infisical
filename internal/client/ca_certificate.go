package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGenerateCACertificate = "CallGenerateCACertificate"
	operationGetCACertificate      = "CallGetCACertificate"
)

func (client Client) GenerateCACertificate(request GenerateCACertificateRequest) (GenerateCACertificateResponse, error) {
	var certResponse GenerateCACertificateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&certResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/ca/internal/%s/certificate", request.CaId))

	if err != nil {
		return GenerateCACertificateResponse{}, errors.NewGenericRequestError(operationGenerateCACertificate, err)
	}

	if response.IsError() {
		return GenerateCACertificateResponse{}, errors.NewAPIErrorWithResponse(operationGenerateCACertificate, response, nil)
	}

	return certResponse, nil
}

func (client Client) GetCACertificate(request GetCACertificateRequest) (GetCACertificateResponse, error) {
	var certResponse GetCACertificateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&certResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/ca/internal/%s/certificate", request.CaId))

	if err != nil {
		return GetCACertificateResponse{}, errors.NewGenericRequestError(operationGetCACertificate, err)
	}

	if response.IsError() {
		return GetCACertificateResponse{}, errors.NewAPIErrorWithResponse(operationGetCACertificate, response, nil)
	}

	return certResponse, nil
}
