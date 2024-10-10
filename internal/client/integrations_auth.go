package infisicalclient

import (
	"fmt"
)

// Enum containing the possible values for the `type` field in the CreateIntegrationAuthRequest.
type IntegrationAuthType string

const (
	IntegrationAuthTypeGcpSecretManager  IntegrationAuthType = "gcp-secret-manager"
	IntegrationAuthTypeAwsParameterStore IntegrationAuthType = "aws-parameter-store"
	IntegrationAuthTypeAwsSecretsManager IntegrationAuthType = "aws-secret-manager"
	IntegrationAuthTypeCircleCi          IntegrationAuthType = "circleci"
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
		return CreateIntegrationAuthResponse{}, fmt.Errorf("CreateIntegrationAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateIntegrationAuthResponse{}, fmt.Errorf("CreateIntegrationAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DeleteIntegrationAuthResponse{}, fmt.Errorf("DeleteIntegrationAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteIntegrationAuthResponse{}, fmt.Errorf("DeleteIntegrationAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil

}
