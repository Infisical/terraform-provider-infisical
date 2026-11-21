package infisicalclient

import (
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetInstanceStatus = "CallGetInstanceStatus"
)

// GetInstanceStatusResponse holds the fields of GET /api/status that the provider uses.
type GetInstanceStatusResponse struct {
	// AwsAccountId is absent when the instance has no AWS identity, and on servers that predate the field.
	AwsAccountId *string `json:"awsAccountId"`
}

func (client Client) GetInstanceStatus() (GetInstanceStatusResponse, error) {
	var body GetInstanceStatusResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/status")

	if err != nil {
		return GetInstanceStatusResponse{}, errors.NewGenericRequestError(operationGetInstanceStatus, err)
	}

	if response.IsError() {
		return GetInstanceStatusResponse{}, errors.NewAPIErrorWithResponse(operationGetInstanceStatus, response, nil)
	}

	return body, nil
}
