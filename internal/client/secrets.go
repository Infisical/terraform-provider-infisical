package infisicalclient

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"terraform-provider-infisical/internal/crypto"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetSecretsV3               = "CallGetSecretsV3"
	operationCreateSecretsV3            = "CallCreateSecretsV3"
	operationDeleteSecretsV3            = "CallDeleteSecretsV3"
	operationUpdateSecretsV3            = "CallUpdateSecretsV3"
	operationGetSingleSecretByNameV3    = "CallGetSingleSecretByNameV3"
	operationGetSingleSecretByIDV3      = "CallGetSingleSecretByIDV3"
	operationGetSecretsRawV3            = "CallGetSecretsRawV3"
	operationCreateRawSecretsV3         = "CallCreateRawSecretsV3"
	operationDeleteRawSecretV3          = "CallDeleteRawSecretV3"
	operationUpdateRawSecretV3          = "CallUpdateRawSecretV3"
	operationGetSingleRawSecretByNameV3 = "CallGetSingleRawSecretByNameV3"
)

func (client Client) GetSecretsV3(request GetEncryptedSecretsV3Request) (GetEncryptedSecretsV3Response, error) {
	var secretsResponse GetEncryptedSecretsV3Response

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("workspaceId", request.WorkspaceId)

	if request.SecretPath != "" {
		httpRequest.SetQueryParam("secretPath", request.SecretPath)
	}

	response, err := httpRequest.Get("api/v3/secrets")

	if err != nil {
		return GetEncryptedSecretsV3Response{}, errors.NewGenericRequestError(operationGetSecretsV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return GetEncryptedSecretsV3Response{}, errors.NewAPIErrorWithResponse(operationGetSecretsV3, response, &additionalContext)
	}

	return secretsResponse, nil
}

func (client Client) CreateSecretsV3(request CreateSecretV3Request) (EncryptedSecretV3, error) {
	var secretsResponse EncryptedSecretV3
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return EncryptedSecretV3{}, errors.NewGenericRequestError(operationCreateSecretsV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return EncryptedSecretV3{}, errors.NewAPIErrorWithResponse(operationCreateSecretsV3, response, &additionalContext)
	}

	return secretsResponse, nil
}

func (client Client) DeleteSecretsV3(request DeleteSecretV3Request) error {
	var secretsResponse GetEncryptedSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteSecretsV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return errors.NewAPIErrorWithResponse(operationDeleteSecretsV3, response, &additionalContext)
	}

	return nil
}

func (client Client) UpdateSecretsV3(request UpdateSecretByNameV3Request) error {

	var secretsResponse GetEncryptedSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return errors.NewGenericRequestError(operationUpdateSecretsV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return errors.NewAPIErrorWithResponse(operationUpdateSecretsV3, response, &additionalContext)
	}

	return nil
}

func (client Client) GetSingleSecretByNameV3(request GetSingleSecretByNameV3Request) (GetSingleSecretByNameSecretResponse, error) {
	var secretsResponse GetSingleSecretByNameSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.WorkspaceId).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("type", request.Type).
		SetQueryParam("secretPath", request.SecretPath).
		Get(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return GetSingleSecretByNameSecretResponse{}, errors.NewGenericRequestError(operationGetSingleSecretByNameV3, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetSingleSecretByNameSecretResponse{}, ErrNotFound
		}
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return GetSingleSecretByNameSecretResponse{}, errors.NewAPIErrorWithResponse(operationGetSingleSecretByNameV3, response, &additionalContext)
	}

	return secretsResponse, nil
}

func (client Client) GetSingleSecretByIDV3(request GetSingleSecretByIDV3Request) (GetSingleSecretByIDV3Response, error) {
	var secretsResponse GetSingleSecretByIDV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v3/secrets/raw/id/%s", request.ID))

	if err != nil {
		return GetSingleSecretByIDV3Response{}, errors.NewGenericRequestError(operationGetSingleSecretByIDV3, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetSingleSecretByIDV3Response{}, ErrNotFound
		}
		return GetSingleSecretByIDV3Response{}, errors.NewAPIErrorWithResponse(operationGetSingleSecretByIDV3, response, nil)
	}

	return secretsResponse, nil
}

func (client Client) GetSecretsRawV3(request GetRawSecretsV3Request) (GetRawSecretsV3Response, error) {
	var secretsResponse GetRawSecretsV3Response

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParams(map[string]string{
			"environment":            request.Environment,
			"workspaceId":            request.WorkspaceId,
			"expandSecretReferences": strconv.FormatBool(request.ExpandSecretReferences),
		})

	if request.SecretPath != "" {
		httpRequest.SetQueryParam("secretPath", request.SecretPath)
	}

	response, err := httpRequest.Get("api/v3/secrets/raw")

	if err != nil {
		return GetRawSecretsV3Response{}, errors.NewGenericRequestError(operationGetSecretsRawV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return GetRawSecretsV3Response{}, errors.NewAPIErrorWithResponse(operationGetSecretsRawV3, response, &additionalContext)
	}

	return secretsResponse, nil
}

func (client Client) CreateRawSecretsV3(request CreateRawSecretV3Request) (RawV3Secret, error) {
	var secretsResponse RawV3Secret
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretKey))

	if err != nil {
		return RawV3Secret{}, errors.NewGenericRequestError(operationCreateRawSecretsV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return RawV3Secret{}, errors.NewAPIErrorWithResponse(operationCreateRawSecretsV3, response, &additionalContext)
	}

	return secretsResponse, nil
}

func (client Client) DeleteRawSecretV3(request DeleteRawSecretV3Request) error {
	var secretsResponse GetRawSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretName))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteRawSecretV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return errors.NewAPIErrorWithResponse(operationDeleteRawSecretV3, response, &additionalContext)
	}

	return nil
}

func (client Client) UpdateRawSecretV3(request UpdateRawSecretByNameV3Request) error {
	var secretsResponse GetRawSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretName))

	if err != nil {
		return errors.NewGenericRequestError(operationUpdateRawSecretV3, err)
	}

	if response.IsError() {
		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return errors.NewAPIErrorWithResponse(operationUpdateRawSecretV3, response, &additionalContext)
	}

	return nil
}

func (client Client) GetSingleRawSecretByNameV3(request GetSingleSecretByNameV3Request) (GetSingleRawSecretByNameSecretResponse, error) {
	var secretsResponse GetSingleRawSecretByNameSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.WorkspaceId).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("type", request.Type).
		SetQueryParam("secretPath", request.SecretPath).
		Get(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretName))

	if err != nil {
		return GetSingleRawSecretByNameSecretResponse{}, errors.NewGenericRequestError(operationGetSingleRawSecretByNameV3, err)
	}

	if response.IsError() {

		if response.StatusCode() == http.StatusNotFound {
			return GetSingleRawSecretByNameSecretResponse{}, ErrNotFound
		}

		additionalContext := "Please make sure your secret path, workspace and environment name are all correct"
		return GetSingleRawSecretByNameSecretResponse{}, errors.NewAPIErrorWithResponse(operationGetSingleRawSecretByNameV3, response, &additionalContext)
	}

	return secretsResponse, nil
}

func (client Client) GetPlainTextSecretsViaServiceToken(secretFolderPath string, envSlug string) ([]SingleEnvironmentVariable, *GetServiceTokenDetailsResponse, error) {
	if client.Config.ServiceToken == "" {
		return nil, nil, fmt.Errorf("service token must be defined to fetch secrets")
	}

	serviceTokenParts := strings.SplitN(client.Config.ServiceToken, ".", 4)
	if len(serviceTokenParts) < 4 {
		return nil, nil, fmt.Errorf("invalid service token entered. Please double check your service token and try again")
	}

	serviceTokenDetails, err := client.GetServiceTokenDetailsV2()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get service token details. [err=%v]", err)
	}

	request := GetEncryptedSecretsV3Request{
		WorkspaceId: serviceTokenDetails.Workspace,
		Environment: envSlug,
	}

	if secretFolderPath != "" {
		request.SecretPath = secretFolderPath
	}

	encryptedSecrets, err := client.GetSecretsV3(request)

	if err != nil {
		return nil, nil, err
	}

	decodedSymmetricEncryptionDetails, err := GetBase64DecodedSymmetricEncryptionDetails(serviceTokenParts[3], serviceTokenDetails.EncryptedKey, serviceTokenDetails.Iv, serviceTokenDetails.Tag)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decode symmetric encryption details [err=%v]", err)
	}

	plainTextWorkspaceKey, err := crypto.DecryptSymmetric([]byte(serviceTokenParts[3]), decodedSymmetricEncryptionDetails.Cipher, decodedSymmetricEncryptionDetails.Tag, decodedSymmetricEncryptionDetails.IV)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decrypt the required workspace key")
	}

	plainTextSecrets, err := GetPlainTextSecrets(plainTextWorkspaceKey, encryptedSecrets)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decrypt your secrets [err=%v]", err)
	}

	return plainTextSecrets, &serviceTokenDetails, nil
}

func (client Client) GetRawSecrets(secretFolderPath string, envSlug string, workspaceId string) ([]RawV3Secret, error) {
	request := GetRawSecretsV3Request{
		Environment:            envSlug,
		WorkspaceId:            workspaceId,
		ExpandSecretReferences: true,
	}

	if secretFolderPath != "" {
		request.SecretPath = secretFolderPath
	}

	secrets, err := client.GetSecretsRawV3(request)

	if err != nil {
		return nil, err
	}

	return secrets.Secrets, nil
}

func GetPlainTextSecrets(key []byte, encryptedSecrets GetEncryptedSecretsV3Response) ([]SingleEnvironmentVariable, error) {
	plainTextSecrets := []SingleEnvironmentVariable{}
	for _, secret := range encryptedSecrets.Secrets {
		// Decrypt key
		key_iv, err := base64.StdEncoding.DecodeString(secret.SecretKeyIV)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret IV for secret key")
		}

		key_tag, err := base64.StdEncoding.DecodeString(secret.SecretKeyTag)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret authentication tag for secret key")
		}

		key_ciphertext, err := base64.StdEncoding.DecodeString(secret.SecretKeyCiphertext)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret cipher text for secret key")
		}

		plainTextKey, err := crypto.DecryptSymmetric(key, key_ciphertext, key_tag, key_iv)
		if err != nil {
			return nil, fmt.Errorf("unable to symmetrically decrypt secret key")
		}

		// Decrypt value
		value_iv, err := base64.StdEncoding.DecodeString(secret.SecretValueIV)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret IV for secret value")
		}

		value_tag, err := base64.StdEncoding.DecodeString(secret.SecretValueTag)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret authentication tag for secret value")
		}

		value_ciphertext, _ := base64.StdEncoding.DecodeString(secret.SecretValueCiphertext)

		plainTextValue, err := crypto.DecryptSymmetric(key, value_ciphertext, value_tag, value_iv)
		if err != nil {
			return nil, fmt.Errorf("unable to symmetrically decrypt secret value")
		}

		// Decrypt comment
		comment_iv, err := base64.StdEncoding.DecodeString(secret.SecretCommentIV)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret IV for secret value")
		}

		comment_tag, err := base64.StdEncoding.DecodeString(secret.SecretCommentTag)
		if err != nil {
			return nil, fmt.Errorf("unable to decode secret authentication tag for secret value")
		}

		comment_ciphertext, _ := base64.StdEncoding.DecodeString(secret.SecretCommentCiphertext)

		plainTextComment, err := crypto.DecryptSymmetric(key, comment_ciphertext, comment_tag, comment_iv)
		if err != nil {
			return nil, fmt.Errorf("unable to symmetrically decrypt secret comment")
		}

		plainTextSecret := SingleEnvironmentVariable{
			Key:     string(plainTextKey),
			Value:   string(plainTextValue),
			Type:    secret.Type,
			ID:      secret.ID,
			Tags:    secret.Tags,
			Comment: string(plainTextComment),
		}

		plainTextSecrets = append(plainTextSecrets, plainTextSecret)
	}

	return plainTextSecrets, nil
}
