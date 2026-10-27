package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePamAccount             = "CallCreatePamAccount"
	operationGetPamAccount                = "CallGetPamAccount"
	operationUpdatePamAccount             = "CallUpdatePamAccount"
	operationDeletePamAccount             = "CallDeletePamAccount"
	operationGetPamAccountRotation        = "CallGetPamAccountRotation"
	operationSetPamAccountRotationAccount = "CallSetPamAccountRotationAccount"
)

func (client Client) CreatePamAccount(request CreatePamAccountRequest) (PamAccount, error) {
	var body PamAccountResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/pam/accounts/%s", request.AccountType))

	if err != nil {
		return PamAccount{}, errors.NewGenericRequestError(operationCreatePamAccount, err)
	}

	if response.IsError() {
		return PamAccount{}, errors.NewAPIErrorWithResponse(operationCreatePamAccount, response, nil)
	}

	return body.Account, nil
}

func (client Client) GetPamAccount(request GetPamAccountRequest) (PamAccount, error) {
	var body PamAccountResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/pam/accounts/%s", request.ID))

	if err != nil {
		return PamAccount{}, errors.NewGenericRequestError(operationGetPamAccount, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return PamAccount{}, ErrNotFound
		}
		return PamAccount{}, errors.NewAPIErrorWithResponse(operationGetPamAccount, response, nil)
	}

	return body.Account, nil
}

func (client Client) UpdatePamAccount(request UpdatePamAccountRequest) (PamAccount, error) {
	var body PamAccountResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request.Body).
		Patch(fmt.Sprintf("api/v1/pam/accounts/%s/%s", request.AccountType, request.ID))

	if err != nil {
		return PamAccount{}, errors.NewGenericRequestError(operationUpdatePamAccount, err)
	}

	if response.IsError() {
		return PamAccount{}, errors.NewAPIErrorWithResponse(operationUpdatePamAccount, response, nil)
	}

	return body.Account, nil
}

func (client Client) DeletePamAccount(request DeletePamAccountRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/pam/accounts/%s/%s", request.AccountType, request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeletePamAccount, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeletePamAccount, response, nil)
	}

	return nil
}

func (client Client) GetPamAccountRotation(request GetPamAccountRotationRequest) (PamAccountRotation, error) {
	var body GetPamAccountRotationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/pam/accounts/%s/rotation", request.AccountID))

	if err != nil {
		return PamAccountRotation{}, errors.NewGenericRequestError(operationGetPamAccountRotation, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return PamAccountRotation{}, ErrNotFound
		}
		return PamAccountRotation{}, errors.NewAPIErrorWithResponse(operationGetPamAccountRotation, response, nil)
	}

	return body.Rotation, nil
}

func (client Client) SetPamAccountRotationAccount(request SetPamAccountRotationAccountRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/pam/accounts/%s/rotation", request.AccountID))

	if err != nil {
		return errors.NewGenericRequestError(operationSetPamAccountRotationAccount, err)
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationSetPamAccountRotationAccount, response, nil)
	}

	return nil
}
