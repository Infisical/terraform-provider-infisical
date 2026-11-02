package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePamAccountTemplate = "CallCreatePamAccountTemplate"
	operationGetPamAccountTemplate    = "CallGetPamAccountTemplate"
	operationUpdatePamAccountTemplate = "CallUpdatePamAccountTemplate"
	operationDeletePamAccountTemplate = "CallDeletePamAccountTemplate"
)

func (client Client) CreatePamAccountTemplate(request CreatePamAccountTemplateRequest) (PamAccountTemplate, error) {
	var body PamAccountTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/pam/account-templates")

	if err != nil {
		return PamAccountTemplate{}, errors.NewGenericRequestError(operationCreatePamAccountTemplate, err)
	}

	if response.IsError() {
		return PamAccountTemplate{}, errors.NewAPIErrorWithResponse(operationCreatePamAccountTemplate, response, nil)
	}

	return body.Template, nil
}

func (client Client) GetPamAccountTemplate(request GetPamAccountTemplateRequest) (PamAccountTemplate, error) {
	var body PamAccountTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/pam/account-templates/%s", request.ID))

	if err != nil {
		return PamAccountTemplate{}, errors.NewGenericRequestError(operationGetPamAccountTemplate, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return PamAccountTemplate{}, ErrNotFound
		}
		return PamAccountTemplate{}, errors.NewAPIErrorWithResponse(operationGetPamAccountTemplate, response, nil)
	}

	return body.Template, nil
}

func (client Client) UpdatePamAccountTemplate(request UpdatePamAccountTemplateRequest) (PamAccountTemplate, error) {
	var body PamAccountTemplateResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/pam/account-templates/%s", request.ID))

	if err != nil {
		return PamAccountTemplate{}, errors.NewGenericRequestError(operationUpdatePamAccountTemplate, err)
	}

	if response.IsError() {
		return PamAccountTemplate{}, errors.NewAPIErrorWithResponse(operationUpdatePamAccountTemplate, response, nil)
	}

	return body.Template, nil
}

func (client Client) DeletePamAccountTemplate(request DeletePamAccountTemplateRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/pam/account-templates/%s", request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeletePamAccountTemplate, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeletePamAccountTemplate, response, nil)
	}

	return nil
}
