package infisicalclient

import (
	"fmt"
)

// Enum containing the possible values for the `type` field in the CreateIntegrationAuthRequest.
type IntegrationAuthType string

const (
	IntegrationAuthTypeGcpSecretManager IntegrationAuthType = "gcp-secret-manager"
)

func (client Client) CallCreateIntegrationAuth(request CreateIntegrationAuthRequest) (CreateIntegrationAuthResponse, error) {
	var body CreateIntegrationAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/integration-auth/access-token")

	if err != nil {
		return CreateIntegrationAuthResponse{}, fmt.Errorf("CallCreateIntegrationAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateIntegrationAuthResponse{}, fmt.Errorf("CallCreateIntegrationAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

// Deleting integration auth triggers a cascade effect, that will also delete the associated integration.
func (client Client) CallDeleteIntegrationAuth(request DeleteIntegrationAuthRequest) (DeleteIntegrationAuthResponse, error) {
	var body DeleteIntegrationAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/integration-auth/%s", request.ID))

	if err != nil {
		return DeleteIntegrationAuthResponse{}, fmt.Errorf("CallDeleteIntegrationAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteIntegrationAuthResponse{}, fmt.Errorf("CallDeleteIntegrationAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil

}
