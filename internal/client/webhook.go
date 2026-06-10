package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateWebhook  = "CallCreateWebhook"
	operationGetWebhookByID = "CallGetWebhookByID"
	operationUpdateWebhook  = "CallUpdateWebhook"
	operationDeleteWebhook  = "CallDeleteWebhook"
)

func (client Client) CreateWebhook(request CreateWebhookRequest) (CreateWebhookResponse, error) {
	var body CreateWebhookResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/webhooks")

	if err != nil {
		return CreateWebhookResponse{}, errors.NewGenericRequestError(operationCreateWebhook, err)
	}

	if response.IsError() {
		return CreateWebhookResponse{}, errors.NewAPIErrorWithResponse(operationCreateWebhook, response, nil)
	}

	return body, nil
}

func (client Client) GetWebhookByID(request GetWebhookByIDRequest) (GetWebhookByIDResponse, error) {
	var body GetWebhookByIDResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/webhooks/" + request.ID)

	if err != nil {
		return GetWebhookByIDResponse{}, errors.NewGenericRequestError(operationGetWebhookByID, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetWebhookByIDResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetWebhookByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetWebhookByID, response, nil)
	}

	return body, nil
}

func (client Client) UpdateWebhook(request UpdateWebhookRequest) (UpdateWebhookResponse, error) {
	var body UpdateWebhookResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/webhooks/%s", request.ID))

	if err != nil {
		return UpdateWebhookResponse{}, errors.NewGenericRequestError(operationUpdateWebhook, err)
	}

	if response.IsError() {
		return UpdateWebhookResponse{}, errors.NewAPIErrorWithResponse(operationUpdateWebhook, response, nil)
	}

	return body, nil
}

func (client Client) DeleteWebhook(request DeleteWebhookRequest) (DeleteWebhookResponse, error) {
	var body DeleteWebhookResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/webhooks/%s", request.ID))

	if err != nil {
		return DeleteWebhookResponse{}, errors.NewGenericRequestError(operationDeleteWebhook, err)
	}

	if response.IsError() {
		return DeleteWebhookResponse{}, errors.NewAPIErrorWithResponse(operationDeleteWebhook, response, nil)
	}

	return body, nil
}
