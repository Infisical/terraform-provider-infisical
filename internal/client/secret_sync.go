package infisicalclient

import (
	"fmt"
	"net/http"
)

type SecretSyncApp string

const (
	SecretSyncAppGCPSecretManager      SecretSyncApp = "gcp-secret-manager"
	SecretSyncAppAWSParameterStore     SecretSyncApp = "aws-parameter-store"
	SecretSyncAppAWSSecretsManager     SecretSyncApp = "aws-secrets-manager"
	SecretSyncAppAzureAppConfiguration SecretSyncApp = "azure-app-configuration"
	SecretSyncAppAzureKeyVault         SecretSyncApp = "azure-key-vault"
	SecretSyncAppGithub                SecretSyncApp = "github"
	SecretSyncApp1Password             SecretSyncApp = "1password"
)

type SecretSyncBehavior string

const (
	SecretSyncBehaviorOverwriteDestination  SecretSyncBehavior = "overwrite-destination"
	SecretSyncBehaviorPrioritizeSource      SecretSyncBehavior = "import-prioritize-source"
	SecretSyncBehaviorPrioritizeDestination SecretSyncBehavior = "import-prioritize-destination"
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
		return SecretSync{}, fmt.Errorf("CreateSecretSync: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return SecretSync{}, fmt.Errorf("CreateSecretSync: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return SecretSync{}, fmt.Errorf("UpdateSecretSync: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return SecretSync{}, fmt.Errorf("UpdateSecretSync: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return SecretSync{}, fmt.Errorf("GetSecretSyncById: Unable to complete api request [err=%s]", err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return SecretSync{}, ErrNotFound
	}

	if response.IsError() {
		return SecretSync{}, fmt.Errorf("GetSecretSyncById: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return SecretSync{}, fmt.Errorf("DeleteSecretSync: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return SecretSync{}, fmt.Errorf("DeleteSecretSync: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.SecretSync, nil
}
