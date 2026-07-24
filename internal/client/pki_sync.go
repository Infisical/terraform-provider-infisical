package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type PkiSyncApp string

const (
	PkiSyncAppAWSCertificateManager PkiSyncApp = "aws-certificate-manager"
)

const (
	operationCreatePkiSync            = "CallCreatePkiSync"
	operationUpdatePkiSync            = "CallUpdatePkiSync"
	operationGetPkiSyncById           = "CallGetPkiSyncById"
	operationDeletePkiSync            = "CallDeletePkiSync"
	operationAddPkiSyncCertificates   = "CallAddPkiSyncCertificates"
	operationListPkiSyncCertificates  = "CallListPkiSyncCertificates"
	operationRemovePkiSyncCertificate = "CallRemovePkiSyncCertificates"
)

// PKI syncs are served under the cert-manager namespace, keyed by destination.
func pkiSyncBaseURL(app PkiSyncApp) string {
	return fmt.Sprintf("api/v1/cert-manager/syncs/%s", string(app))
}

func (client Client) CreatePkiSync(request CreatePkiSyncRequest) (PkiSync, error) {
	var body PkiSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(pkiSyncBaseURL(request.App))

	if err != nil {
		return PkiSync{}, errors.NewGenericRequestError(operationCreatePkiSync, err)
	}

	if response.IsError() {
		return PkiSync{}, errors.NewAPIErrorWithResponse(operationCreatePkiSync, response, nil)
	}

	return body, nil
}

func (client Client) UpdatePkiSync(request UpdatePkiSyncRequest) (PkiSync, error) {
	var body PkiSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("%s/%s", pkiSyncBaseURL(request.App), request.ID))

	if err != nil {
		return PkiSync{}, errors.NewGenericRequestError(operationUpdatePkiSync, err)
	}

	if response.IsError() {
		return PkiSync{}, errors.NewAPIErrorWithResponse(operationUpdatePkiSync, response, nil)
	}

	return body, nil
}

func (client Client) GetPkiSyncById(request GetPkiSyncByIdRequest) (PkiSync, error) {
	var body PkiSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/syncs/%s", request.ID))

	if err != nil {
		return PkiSync{}, errors.NewGenericRequestError(operationGetPkiSyncById, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return PkiSync{}, ErrNotFound
	}

	if response.IsError() {
		return PkiSync{}, errors.NewAPIErrorWithResponse(operationGetPkiSyncById, response, nil)
	}

	return body, nil
}

func (client Client) DeletePkiSync(request DeletePkiSyncRequest) (PkiSync, error) {
	var body PkiSync
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s", pkiSyncBaseURL(request.App), request.ID))

	if err != nil {
		return PkiSync{}, errors.NewGenericRequestError(operationDeletePkiSync, err)
	}

	if response.IsError() {
		return PkiSync{}, errors.NewAPIErrorWithResponse(operationDeletePkiSync, response, nil)
	}

	return body, nil
}

// Certificate association endpoints live on the shared sync router, so they are keyed by
// sync ID only (destination-agnostic): api/v1/cert-manager/syncs/:pkiSyncId/certificates.
func pkiSyncCertificatesURL(pkiSyncID string) string {
	return fmt.Sprintf("api/v1/cert-manager/syncs/%s/certificates", pkiSyncID)
}

func (client Client) AddPkiSyncCertificates(request AddPkiSyncCertificatesRequest) ([]PkiSyncCertificate, error) {
	var body AddPkiSyncCertificatesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(pkiSyncCertificatesURL(request.PkiSyncID))

	if err != nil {
		return nil, errors.NewGenericRequestError(operationAddPkiSyncCertificates, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationAddPkiSyncCertificates, response, nil)
	}

	return body.AddedCertificates, nil
}

func (client Client) ListPkiSyncCertificates(request ListPkiSyncCertificatesRequest) (ListPkiSyncCertificatesResponse, error) {
	var body ListPkiSyncCertificatesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParams(map[string]string{
			"offset": fmt.Sprintf("%d", request.Offset),
			"limit":  fmt.Sprintf("%d", request.Limit),
		}).
		Get(pkiSyncCertificatesURL(request.PkiSyncID))

	if err != nil {
		return ListPkiSyncCertificatesResponse{}, errors.NewGenericRequestError(operationListPkiSyncCertificates, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ListPkiSyncCertificatesResponse{}, ErrNotFound
	}

	if response.IsError() {
		return ListPkiSyncCertificatesResponse{}, errors.NewAPIErrorWithResponse(operationListPkiSyncCertificates, response, nil)
	}

	return body, nil
}

func (client Client) RemovePkiSyncCertificates(request RemovePkiSyncCertificatesRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(pkiSyncCertificatesURL(request.PkiSyncID))

	if err != nil {
		return errors.NewGenericRequestError(operationRemovePkiSyncCertificate, err)
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationRemovePkiSyncCertificate, response, nil)
	}

	return nil
}
