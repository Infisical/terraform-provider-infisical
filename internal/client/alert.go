package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateAlert  = "CallCreateAlert"
	operationListAlerts   = "CallListAlerts"
	operationGetAlertByID = "CallGetAlertByID"
	operationUpdateAlert  = "CallUpdateAlert"
	operationDeleteAlert  = "CallDeleteAlert"
)

type AlertAlreadyExistsError struct {
	ExistingAlertID string
	apiError        error
}

func (e *AlertAlreadyExistsError) Error() string {
	return fmt.Sprintf("alert %s already watches this resource for this event: %v", e.ExistingAlertID, e.apiError)
}

func (e *AlertAlreadyExistsError) Unwrap() error {
	return e.apiError
}

func (client Client) existingAlertForEvent(request CreateAlertRequest) *Alert {
	alerts, err := client.ListAlerts(ListAlertsRequest{
		ResourceType: request.ResourceType,
		ResourceID:   request.ResourceID,
		ProjectID:    request.ProjectID,
	})
	if err != nil {
		return nil
	}

	for _, alert := range alerts.Alerts {
		if alert.EventType == request.EventType {
			return &alert
		}
	}

	return nil
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
		if response.StatusCode() == http.StatusBadRequest {
			if existing := client.existingAlertForEvent(request); existing != nil {
				return CreateAlertResponse{}, &AlertAlreadyExistsError{ExistingAlertID: existing.ID, apiError: apiError}
			}
		}
		return CreateAlertResponse{}, apiError
	}

	return body, nil
}

func (client Client) ListAlerts(request ListAlertsRequest) (ListAlertsResponse, error) {
	var body ListAlertsResponse
	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("resourceType", request.ResourceType)

	if request.ResourceID != "" {
		httpRequest.SetQueryParam("resourceId", request.ResourceID)
	}
	if request.ProjectID != nil {
		httpRequest.SetQueryParam("projectId", *request.ProjectID)
	}

	response, err := httpRequest.Get("api/v1/alerts")

	if err != nil {
		return ListAlertsResponse{}, errors.NewGenericRequestError(operationListAlerts, err)
	}

	if response.IsError() {
		return ListAlertsResponse{}, errors.NewAPIErrorWithResponse(operationListAlerts, response, nil)
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
