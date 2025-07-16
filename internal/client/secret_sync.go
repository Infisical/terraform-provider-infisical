package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type SecretSyncApp string

const (
	SecretSyncAppGCPSecretManager      SecretSyncApp = "gcp-secret-manager"
	SecretSyncAppAWSParameterStore     SecretSyncApp = "aws-parameter-store"
	SecretSyncAppAWSSecretsManager     SecretSyncApp = "aws-secrets-manager"
	SecretSyncAppAzureAppConfiguration SecretSyncApp = "azure-app-configuration"
	SecretSyncAppAzureKeyVault         SecretSyncApp = "azure-key-vault"
	SecretSyncAppAzureDevOps           SecretSyncApp = "azure-devops"
	SecretSyncAppGithub                SecretSyncApp = "github"
)

type SecretSyncBehavior string

const (
	SecretSyncBehaviorOverwriteDestination  SecretSyncBehavior = "overwrite-destination"
	SecretSyncBehaviorPrioritizeSource      SecretSyncBehavior = "import-prioritize-source"
	SecretSyncBehaviorPrioritizeDestination SecretSyncBehavior = "import-prioritize-destination"
)

const (
	operationCreateSecretSync  = "CallCreateSecretSync"
	operationUpdateSecretSync  = "CallUpdateSecretSync"
	operationGetSecretSyncById = "CallGetSecretSyncById"
	operationDeleteSecretSync  = "CallDeleteSecretSync"
)

func (client Client) CreateSecretSync(request CreateSecretSyncRequest) (SecretSync, error) {
	var body CreateSecretSyncResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/secret-syncs/" + string(request.App))

	if err != nil {
		return SecretSync{}, errors.NewGenericRequestError(operationCreateSecretSync, err)
	}

	if response.IsError() {
		return SecretSync{}, errors.NewAPIErrorWithResponse(operationCreateSecretSync, response, nil)
	}

	return body.SecretSync, nil
}

func (client Client) UpdateSecretSync(request UpdateSecretSyncRequest) (SecretSync, error) {
	var body UpdateSecretSyncResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/secret-syncs/%s/%s", string(request.App), request.ID))

	if err != nil {
		return SecretSync{}, errors.NewGenericRequestError(operationUpdateSecretSync, err)
	}

	if response.IsError() {
		return SecretSync{}, errors.NewAPIErrorWithResponse(operationUpdateSecretSync, response, nil)
	}

	return body.SecretSync, nil
}

func (client Client) GetSecretSyncById(request GetSecretSyncByIdRequest) (SecretSync, error) {
	var body GetSecretSyncByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/secret-syncs/%s/%s", string(request.App), request.ID))

	if err != nil {
		return SecretSync{}, errors.NewGenericRequestError(operationGetSecretSyncById, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return SecretSync{}, ErrNotFound
	}

	if response.IsError() {
		return SecretSync{}, errors.NewAPIErrorWithResponse(operationGetSecretSyncById, response, nil)
	}

	return body.SecretSync, nil
}

func (client Client) DeleteSecretSync(request DeleteSecretSyncRequest) (SecretSync, error) {
	var body DeleteSecretSyncResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/secret-syncs/%s/%s", string(request.App), request.ID))

	if err != nil {
		return SecretSync{}, errors.NewGenericRequestError(operationDeleteSecretSync, err)
	}

	if response.IsError() {
		return SecretSync{}, errors.NewAPIErrorWithResponse(operationDeleteSecretSync, response, nil)
	}

	return body.SecretSync, nil
}
