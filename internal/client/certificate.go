package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationRequestCertificate          = "CallRequestCertificate"
	operationGetCertificate              = "CallGetCertificate"
	operationGetCertificateRequestStatus = "CallGetCertificateRequestStatus"
	operationRenewCertificate            = "CallRenewCertificate"
	operationRevokeCertificate           = "CallRevokeCertificate"
	operationDeleteCertificate           = "CallDeleteCertificate"
	operationGetCertificateBundle        = "CallGetCertificateBundle"
	operationGetCertificateBody          = "CallGetCertificateBody"
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
	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&certResponse).
		SetHeader("User-Agent", USER_AGENT)
	if request.IncludeDeleted {
		httpRequest.SetQueryParam("includeDeleted", "true")
	}
	response, err := httpRequest.Get(fmt.Sprintf("api/v1/cert-manager/certificates/%s", request.CertificateId))

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

func (client Client) RenewCertificate(request RenewCertificateRequest) (RenewCertificateResponse, error) {
	var renewResponse RenewCertificateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&renewResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/certificates/%s/renew", request.CertificateId))

	if err != nil {
		return RenewCertificateResponse{}, errors.NewGenericRequestError(operationRenewCertificate, err)
	}

	if response.IsError() {
		return RenewCertificateResponse{}, errors.NewAPIErrorWithResponse(operationRenewCertificate, response, nil)
	}

	return renewResponse, nil
}

func (client Client) RevokeCertificate(request RevokeCertificateRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/certificates/%s/revoke", request.CertificateId))

	if err != nil {
		return errors.NewGenericRequestError(operationRevokeCertificate, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationRevokeCertificate, response, nil)
	}

	return nil
}

func (client Client) DeleteCertificate(request DeleteCertificateRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/certificates/%s", request.CertificateId))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteCertificate, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeleteCertificate, response, nil)
	}

	return nil
}

func (client Client) GetCertificateBundle(request GetCertificateRequest) (GetCertificateBundleResponse, error) {
	var bundleResponse GetCertificateBundleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&bundleResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificates/%s/bundle", request.CertificateId))

	if err != nil {
		return GetCertificateBundleResponse{}, errors.NewGenericRequestError(operationGetCertificateBundle, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 {
			return GetCertificateBundleResponse{}, ErrNotFound
		}
		return GetCertificateBundleResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificateBundle, response, nil)
	}

	return bundleResponse, nil
}

func (client Client) GetCertificateBody(request GetCertificateRequest) (GetCertificateBodyResponse, error) {
	var bodyResponse GetCertificateBodyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&bodyResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificates/%s/certificate", request.CertificateId))

	if err != nil {
		return GetCertificateBodyResponse{}, errors.NewGenericRequestError(operationGetCertificateBody, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 {
			return GetCertificateBodyResponse{}, ErrNotFound
		}
		return GetCertificateBodyResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificateBody, response, nil)
	}

	return bodyResponse, nil
}
