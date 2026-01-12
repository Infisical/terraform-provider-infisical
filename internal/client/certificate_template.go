package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateCertificateTemplate = "CallCreateCertificateTemplate"
	operationGetCertificateTemplate    = "CallGetCertificateTemplate"
	operationUpdateCertificateTemplate = "CallUpdateCertificateTemplate"
	operationDeleteCertificateTemplate = "CallDeleteCertificateTemplate"
)

func (client Client) CreateCertificateTemplate(request CreateCertificateTemplateRequest) (CreateCertificateTemplateResponse, error) {
	var templateResponse CreateCertificateTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&templateResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/certificate-templates")

	if err != nil {
		return CreateCertificateTemplateResponse{}, errors.NewGenericRequestError(operationCreateCertificateTemplate, err)
	}

	if response.IsError() {
		return CreateCertificateTemplateResponse{}, errors.NewAPIErrorWithResponse(operationCreateCertificateTemplate, response, nil)
	}

	return templateResponse, nil
}

func (client Client) GetCertificateTemplate(request GetCertificateTemplateRequest) (GetCertificateTemplateResponse, error) {
	var templateResponse GetCertificateTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&templateResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/certificate-templates/%s", request.TemplateId))

	if err != nil {
		return GetCertificateTemplateResponse{}, errors.NewGenericRequestError(operationGetCertificateTemplate, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertificateTemplateResponse{}, ErrNotFound
		}
		return GetCertificateTemplateResponse{}, errors.NewAPIErrorWithResponse(operationGetCertificateTemplate, response, nil)
	}

	return templateResponse, nil
}

func (client Client) UpdateCertificateTemplate(request UpdateCertificateTemplateRequest) (UpdateCertificateTemplateResponse, error) {
	var templateResponse UpdateCertificateTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&templateResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/certificate-templates/%s", request.TemplateId))

	if err != nil {
		return UpdateCertificateTemplateResponse{}, errors.NewGenericRequestError(operationUpdateCertificateTemplate, err)
	}

	if response.IsError() {
		return UpdateCertificateTemplateResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertificateTemplate, response, nil)
	}

	return templateResponse, nil
}

func (client Client) DeleteCertificateTemplate(request DeleteCertificateTemplateRequest) (DeleteCertificateTemplateResponse, error) {
	var templateResponse DeleteCertificateTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&templateResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/certificate-templates/%s", request.TemplateId))

	if err != nil {
		return DeleteCertificateTemplateResponse{}, errors.NewGenericRequestError(operationDeleteCertificateTemplate, err)
	}

	if response.IsError() {
		return DeleteCertificateTemplateResponse{}, errors.NewAPIErrorWithResponse(operationDeleteCertificateTemplate, response, nil)
	}

	return templateResponse, nil
}
