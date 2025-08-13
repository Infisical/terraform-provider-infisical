package infisicalclient

import (
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityDetails = "CallGetIdentityDetails"
)

func (client Client) GetIdentityDetails() (GetIdentityDetailsResponse, error) {
	var body IdentityDetails

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/organization")

	if err != nil {
		return GetIdentityDetailsResponse{}, errors.NewGenericRequestError(operationGetIdentityDetails, err)
	}

	if response.IsError() {
		return GetIdentityDetailsResponse{}, errors.NewAPIErrorWithResponse(operationGetIdentityDetails, response, nil)
	}

	// Wrap the response in the expected structure
	return GetIdentityDetailsResponse{
		IdentityDetails: body,
	}, nil
}
