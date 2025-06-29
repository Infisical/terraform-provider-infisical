package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

// Enum containing the possible values for the `type` field in the CreateIntegrationAuthRequest.
type IntegrationAuthType string

const (
	IntegrationAuthTypeGcpSecretManager  IntegrationAuthType = "gcp-secret-manager"
	IntegrationAuthTypeAwsParameterStore IntegrationAuthType = "aws-parameter-store"
	IntegrationAuthTypeDatabricks        IntegrationAuthType = "databricks"
	IntegrationAuthTypeAwsSecretsManager IntegrationAuthType = "aws-secret-manager"
	IntegrationAuthTypeCircleCi          IntegrationAuthType = "circleci"
)

const (
	operationCreateIntegrationAuth = "CallCreateIntegrationAuth"
	operationUpdateIntegrationAuth = "CallUpdateIntegrationAuth"
	operationDeleteIntegrationAuth = "CallDeleteIntegrationAuth"
)

func (client Client) CreateIntegrationAuth(request CreateIntegrationAuthRequest) (CreateIntegrationAuthResponse, error) {
	var body CreateIntegrationAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/integration-auth/access-token")

	if err != nil {
		return CreateIntegrationAuthResponse{}, errors.NewGenericRequestError(operationCreateIntegrationAuth, err)
	}

	if response.IsError() {
		return CreateIntegrationAuthResponse{}, errors.NewAPIErrorWithResponse(operationCreateIntegrationAuth, response, nil)
	}

	return body, nil
}

func (client Client) UpdateIntegrationAuth(request UpdateIntegrationAuthRequest) (UpdateIntegrationAuthResponse, error) {
	var body UpdateIntegrationAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/integration-auth/" + request.IntegrationAuthId)

	if err != nil {
		return UpdateIntegrationAuthResponse{}, errors.NewGenericRequestError(operationUpdateIntegrationAuth, err)
	}

	if response.IsError() {
		return UpdateIntegrationAuthResponse{}, errors.NewAPIErrorWithResponse(operationUpdateIntegrationAuth, response, nil)
	}

	return body, nil
}

// Deleting integration auth triggers a cascade effect, that will also delete the associated integration.
func (client Client) DeleteIntegrationAuth(request DeleteIntegrationAuthRequest) (DeleteIntegrationAuthResponse, error) {
	var body DeleteIntegrationAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/integration-auth/%s", request.ID))

	if err != nil {
		return DeleteIntegrationAuthResponse{}, errors.NewGenericRequestError(operationDeleteIntegrationAuth, err)
	}

	if response.IsError() {
		return DeleteIntegrationAuthResponse{}, errors.NewAPIErrorWithResponse(operationDeleteIntegrationAuth, response, nil)
	}

	return body, nil

}
