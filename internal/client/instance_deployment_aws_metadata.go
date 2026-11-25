package infisicalclient

import (
	"net/http"

	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetInstanceDeploymentAwsMetadata = "CallGetInstanceDeploymentAwsMetadata"
)

type InstanceDeploymentAwsMetadata struct {
	// AccountId is null when the instance has no AWS identity.
	AccountId *string `json:"accountId"`
}

type GetInstanceDeploymentAwsMetadataResponse struct {
	Metadata InstanceDeploymentAwsMetadata `json:"metadata"`
}

// GetInstanceDeploymentAwsMetadata returns ErrNotFound when the Infisical server predates the endpoint.
func (client Client) GetInstanceDeploymentAwsMetadata() (InstanceDeploymentAwsMetadata, error) {
	var body GetInstanceDeploymentAwsMetadataResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/instance/deployment/aws/metadata")

	if err != nil {
		return InstanceDeploymentAwsMetadata{}, errors.NewGenericRequestError(operationGetInstanceDeploymentAwsMetadata, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return InstanceDeploymentAwsMetadata{}, ErrNotFound
		}
		return InstanceDeploymentAwsMetadata{}, errors.NewAPIErrorWithResponse(operationGetInstanceDeploymentAwsMetadata, response, nil)
	}

	return body.Metadata, nil
}
