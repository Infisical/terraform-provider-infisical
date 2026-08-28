package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetSecretApprovalRequestByID = "CallGetSecretApprovalRequestByID"
)

func (client Client) GetSecretApprovalRequestByID(request GetSecretApprovalRequestByIDRequest) (GetSecretApprovalRequestByIDResponse, error) {
	var body GetSecretApprovalRequestByIDResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/secret-approval-requests/%s", request.ID))

	if err != nil {
		return GetSecretApprovalRequestByIDResponse{}, errors.NewGenericRequestError(operationGetSecretApprovalRequestByID, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetSecretApprovalRequestByIDResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetSecretApprovalRequestByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretApprovalRequestByID, response, nil)
	}

	return body, nil
}
