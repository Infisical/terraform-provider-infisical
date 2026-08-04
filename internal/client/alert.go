package infisicalclient

import (
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-infisical/internal/errors"

	"github.com/go-resty/resty/v2"
)

const (
	operationCreateAlert  = "CallCreateAlert"
	operationGetAlertByID = "CallGetAlertByID"
	operationUpdateAlert  = "CallUpdateAlert"
	operationDeleteAlert  = "CallDeleteAlert"
)

func isDuplicateAlertResponse(response *resty.Response) bool {
	return response.StatusCode() == http.StatusBadRequest && strings.Contains(response.String(), "already exists")
}

func (client Client) CreateAlert(request CreateAlertRequest) (CreateAlertResponse, error) {
	var body CreateAlertResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/alerts")

	if err != nil {
		return CreateAlertResponse{}, errors.NewGenericRequestError(operationCreateAlert, err)
	}

	if response.IsError() {
		apiError := errors.NewAPIErrorWithResponse(operationCreateAlert, response, nil)
		if isDuplicateAlertResponse(response) {
			return CreateAlertResponse{}, fmt.Errorf("%w: %w", ErrAlertAlreadyExists, apiError)
		}
		return CreateAlertResponse{}, apiError
	}

	return body, nil
}

func (client Client) GetAlertByID(request GetAlertByIDRequest) (GetAlertByIDResponse, error) {
	var body GetAlertByIDResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/alerts/%s", request.ID))

	if err != nil {
		return GetAlertByIDResponse{}, errors.NewGenericRequestError(operationGetAlertByID, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound || response.StatusCode() == http.StatusUnprocessableEntity {
			return GetAlertByIDResponse{}, ErrNotFound
		}
		return GetAlertByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetAlertByID, response, nil)
	}

	return body, nil
}

func (client Client) UpdateAlert(request UpdateAlertRequest) (UpdateAlertResponse, error) {
	var body UpdateAlertResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/alerts/%s", request.ID))

	if err != nil {
		return UpdateAlertResponse{}, errors.NewGenericRequestError(operationUpdateAlert, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound || response.StatusCode() == http.StatusUnprocessableEntity {
			return UpdateAlertResponse{}, ErrNotFound
		}
		return UpdateAlertResponse{}, errors.NewAPIErrorWithResponse(operationUpdateAlert, response, nil)
	}

	return body, nil
}

func (client Client) DeleteAlert(request DeleteAlertRequest) (DeleteAlertResponse, error) {
	var body DeleteAlertResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/alerts/%s", request.ID))

	if err != nil {
		return DeleteAlertResponse{}, errors.NewGenericRequestError(operationDeleteAlert, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound || response.StatusCode() == http.StatusUnprocessableEntity {
			return DeleteAlertResponse{}, ErrNotFound
		}
		return DeleteAlertResponse{}, errors.NewAPIErrorWithResponse(operationDeleteAlert, response, nil)
	}

	return body, nil
}
