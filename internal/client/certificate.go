package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationRequestCertificate          = "CallRequestCertificate"
	operationGetCertificate              = "CallGetCertificate"
	operationGetCertificateRequestStatus = "CallGetCertificateRequestStatus"
)

func (client Client) RequestCertificate(request RequestCertificateRequest) (RequestCertificateResponse, error) {
	var certResponse RequestCertificateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&certResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/certificates")

	if err != nil {
		return RequestCertificateResponse{}, errors.NewGenericRequestError(operationRequestCertificate, err)
	}

	if response.IsError() {
		return RequestCertificateResponse{}, errors.NewAPIErrorWithResponse(operationRequestCertificate, response, nil)
	}

	return certResponse, nil
}

func (client Client) GetCertificate(request GetCertificateRequest) (GetCertificateResponse, error) {
	var certResponse GetCertificateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&certResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificates/%s", request.CertificateId))

	if err != nil {
		return GetCertificateResponse{}, errors.NewGenericRequestError(operationGetCertificate, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertificateResponse{}, ErrNotFound
		}
		return GetCertificateResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificate, response, nil)
	}

	return certResponse, nil
}

func (client Client) GetCertificateRequestStatus(request GetCertificateRequestStatusRequest) (GetCertificateRequestStatusResponse, error) {
	var statusResponse GetCertificateRequestStatusResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&statusResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificates/certificate-requests/%s", request.RequestId))

	if err != nil {
		return GetCertificateRequestStatusResponse{}, errors.NewGenericRequestError(operationGetCertificateRequestStatus, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertificateRequestStatusResponse{}, ErrNotFound
		}
		return GetCertificateRequestStatusResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificateRequestStatus, response, nil)
	}

	return statusResponse, nil
}
