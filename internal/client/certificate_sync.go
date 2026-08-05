package infisicalclient

import (
	"fmt"
	"net/http"
	"net/url"
	"terraform-provider-infisical/internal/errors"
)

type CertificateSyncApp string

const (
	CertificateSyncAppAWSCertificateManager CertificateSyncApp = "aws-certificate-manager"
)

const (
	operationCreateCertificateSync            = "CallCreateCertificateSync"
	operationUpdateCertificateSync            = "CallUpdateCertificateSync"
	operationGetCertificateSyncById           = "CallGetCertificateSyncById"
	operationDeleteCertificateSync            = "CallDeleteCertificateSync"
	operationAddCertificateSyncCertificates   = "CallAddCertificateSyncCertificates"
	operationListCertificateSyncCertificates  = "CallListCertificateSyncCertificates"
	operationRemoveCertificateSyncCertificate = "CallRemoveCertificateSyncCertificates"
)

// Each sync destination gets its own URL prefix, so the destination is part of the path.
func certificateSyncBaseURL(app CertificateSyncApp) string {
	return fmt.Sprintf("api/v1/cert-manager/syncs/%s", string(app))
}

func (client Client) CreateCertificateSync(request CreateCertificateSyncRequest) (CertificateSync, error) {
	var body CertificateSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(certificateSyncBaseURL(request.App))

	if err != nil {
		return CertificateSync{}, errors.NewGenericRequestError(operationCreateCertificateSync, err)
	}

	if response.IsError() {
		return CertificateSync{}, errors.NewAPIErrorWithResponse(operationCreateCertificateSync, response, nil)
	}

	return body, nil
}

func (client Client) UpdateCertificateSync(request UpdateCertificateSyncRequest) (CertificateSync, error) {
	var body CertificateSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("%s/%s", certificateSyncBaseURL(request.App), url.PathEscape(request.ID)))

	if err != nil {
		return CertificateSync{}, errors.NewGenericRequestError(operationUpdateCertificateSync, err)
	}

	if response.IsError() {
		return CertificateSync{}, errors.NewAPIErrorWithResponse(operationUpdateCertificateSync, response, nil)
	}

	return body, nil
}

func (client Client) GetCertificateSyncById(request GetCertificateSyncByIdRequest) (CertificateSync, error) {
	var body CertificateSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/syncs/%s", url.PathEscape(request.ID)))

	if err != nil {
		return CertificateSync{}, errors.NewGenericRequestError(operationGetCertificateSyncById, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return CertificateSync{}, ErrNotFound
	}

	if response.IsError() {
		return CertificateSync{}, errors.NewAPIErrorWithResponse(operationGetCertificateSyncById, response, nil)
	}

	return body, nil
}

func (client Client) DeleteCertificateSync(request DeleteCertificateSyncRequest) (CertificateSync, error) {
	var body CertificateSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s", certificateSyncBaseURL(request.App), url.PathEscape(request.ID)))

	if err != nil {
		return CertificateSync{}, errors.NewGenericRequestError(operationDeleteCertificateSync, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return CertificateSync{}, nil
	}

	if response.IsError() {
		return CertificateSync{}, errors.NewAPIErrorWithResponse(operationDeleteCertificateSync, response, nil)
	}

	return body, nil
}

// Attaching and detaching certificates works the same way for every destination, so unlike the
// endpoints above these are addressed by sync ID alone and the path carries no destination.
func certificateSyncCertificatesURL(certificateSyncID string) string {
	return fmt.Sprintf("api/v1/cert-manager/syncs/%s/certificates", url.PathEscape(certificateSyncID))
}

func (client Client) AddCertificateSyncCertificates(request AddCertificateSyncCertificatesRequest) ([]CertificateSyncCertificate, error) {
	var body AddCertificateSyncCertificatesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(certificateSyncCertificatesURL(request.CertificateSyncID))

	if err != nil {
		return nil, errors.NewGenericRequestError(operationAddCertificateSyncCertificates, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationAddCertificateSyncCertificates, response, nil)
	}

	return body.AddedCertificates, nil
}

func (client Client) ListCertificateSyncCertificates(request ListCertificateSyncCertificatesRequest) (ListCertificateSyncCertificatesResponse, error) {
	var body ListCertificateSyncCertificatesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParams(map[string]string{
			"offset": fmt.Sprintf("%d", request.Offset),
			"limit":  fmt.Sprintf("%d", request.Limit),
		}).
		Get(certificateSyncCertificatesURL(request.CertificateSyncID))

	if err != nil {
		return ListCertificateSyncCertificatesResponse{}, errors.NewGenericRequestError(operationListCertificateSyncCertificates, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ListCertificateSyncCertificatesResponse{}, ErrNotFound
	}

	if response.IsError() {
		return ListCertificateSyncCertificatesResponse{}, errors.NewAPIErrorWithResponse(operationListCertificateSyncCertificates, response, nil)
	}

	return body, nil
}

func (client Client) RemoveCertificateSyncCertificates(request RemoveCertificateSyncCertificatesRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(certificateSyncCertificatesURL(request.CertificateSyncID))

	if err != nil {
		return errors.NewGenericRequestError(operationRemoveCertificateSyncCertificate, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return nil
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationRemoveCertificateSyncCertificate, response, nil)
	}

	return nil
}
