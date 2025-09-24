package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateKMSKey               = "CallCreateKMSKey"
	operationGetKMSKey                  = "CallGetKMSKey"
	operationGetKMSKeyByName            = "CallGetKMSKeyByName"
	operationListKMSKeys                = "CallListKMSKeys"
	operationUpdateKMSKey               = "CallUpdateKMSKey"
	operationDeleteKMSKey               = "CallDeleteKMSKey"
	operationGetKMSKeyPublicKey         = "CallGetKMSKeyPublicKey"
	operationGetKMSKeySigningAlgorithms = "CallGetKMSKeySigningAlgorithms"
)

func (client Client) CreateKMSKey(request CreateKMSKeyRequest) (CreateKMSKeyResponse, error) {
	var kmsKeyResponse CreateKMSKeyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&kmsKeyResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/kms/keys")

	if err != nil {
		return CreateKMSKeyResponse{}, errors.NewGenericRequestError(operationCreateKMSKey, err)
	}

	if response.IsError() {
		return CreateKMSKeyResponse{}, errors.NewAPIErrorWithResponse(operationCreateKMSKey, response, nil)
	}

	return kmsKeyResponse, nil
}

func (client Client) GetKMSKey(request GetKMSKeyRequest) (GetKMSKeyResponse, error) {
	var kmsKeyResponse GetKMSKeyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&kmsKeyResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/kms/keys/%s", request.KeyId))

	if err != nil {
		return GetKMSKeyResponse{}, errors.NewGenericRequestError(operationGetKMSKey, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetKMSKeyResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetKMSKeyResponse{}, errors.NewAPIErrorWithResponse(operationGetKMSKey, response, nil)
	}

	return kmsKeyResponse, nil
}

func (client Client) GetKMSKeyByName(request GetKMSKeyByNameRequest) (GetKMSKeyByNameResponse, error) {
	var kmsKeyResponse GetKMSKeyByNameResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&kmsKeyResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId).
		Get(fmt.Sprintf("api/v1/kms/keys/key-name/%s", request.KeyName))

	if err != nil {
		return GetKMSKeyByNameResponse{}, errors.NewGenericRequestError(operationGetKMSKeyByName, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetKMSKeyByNameResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetKMSKeyByNameResponse{}, errors.NewAPIErrorWithResponse(operationGetKMSKeyByName, response, nil)
	}

	return kmsKeyResponse, nil
}

func (client Client) ListKMSKeys(request ListKMSKeysRequest) (ListKMSKeysResponse, error) {
	var kmsKeysResponse ListKMSKeysResponse
	req := client.Config.HttpClient.
		R().
		SetResult(&kmsKeysResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectId)

	if request.Offset != nil {
		req.SetQueryParam("offset", fmt.Sprintf("%d", *request.Offset))
	}
	if request.Limit != nil {
		req.SetQueryParam("limit", fmt.Sprintf("%d", *request.Limit))
	}
	if request.OrderBy != nil {
		req.SetQueryParam("orderBy", *request.OrderBy)
	}
	if request.OrderDirection != nil {
		req.SetQueryParam("orderDirection", *request.OrderDirection)
	}
	if request.Search != nil {
		req.SetQueryParam("search", *request.Search)
	}

	response, err := req.Get("api/v1/kms/keys")

	if err != nil {
		return ListKMSKeysResponse{}, errors.NewGenericRequestError(operationListKMSKeys, err)
	}

	if response.IsError() {
		return ListKMSKeysResponse{}, errors.NewAPIErrorWithResponse(operationListKMSKeys, response, nil)
	}

	return kmsKeysResponse, nil
}

func (client Client) UpdateKMSKey(request UpdateKMSKeyRequest) (UpdateKMSKeyResponse, error) {
	var kmsKeyResponse UpdateKMSKeyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&kmsKeyResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/kms/keys/%s", request.KeyId))

	if err != nil {
		return UpdateKMSKeyResponse{}, errors.NewGenericRequestError(operationUpdateKMSKey, err)
	}

	if response.IsError() {
		return UpdateKMSKeyResponse{}, errors.NewAPIErrorWithResponse(operationUpdateKMSKey, response, nil)
	}

	return kmsKeyResponse, nil
}

func (client Client) DeleteKMSKey(request DeleteKMSKeyRequest) (DeleteKMSKeyResponse, error) {
	var kmsKeyResponse DeleteKMSKeyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&kmsKeyResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/kms/keys/%s", request.KeyId))

	if err != nil {
		return DeleteKMSKeyResponse{}, errors.NewGenericRequestError(operationDeleteKMSKey, err)
	}

	if response.IsError() {
		return DeleteKMSKeyResponse{}, errors.NewAPIErrorWithResponse(operationDeleteKMSKey, response, nil)
	}

	return kmsKeyResponse, nil
}

func (client Client) GetKMSKeyPublicKey(request GetKMSKeyPublicKeyRequest) (GetKMSKeyPublicKeyResponse, error) {
	var publicKeyResponse GetKMSKeyPublicKeyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&publicKeyResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/kms/keys/%s/public-key", request.KeyId))

	if err != nil {
		return GetKMSKeyPublicKeyResponse{}, errors.NewGenericRequestError(operationGetKMSKeyPublicKey, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetKMSKeyPublicKeyResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetKMSKeyPublicKeyResponse{}, errors.NewAPIErrorWithResponse(operationGetKMSKeyPublicKey, response, nil)
	}

	return publicKeyResponse, nil
}

func (client Client) GetKMSKeySigningAlgorithms(request GetKMSKeySigningAlgorithmsRequest) (GetKMSKeySigningAlgorithmsResponse, error) {
	var signingAlgorithmsResponse GetKMSKeySigningAlgorithmsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&signingAlgorithmsResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/kms/keys/%s/signing-algorithms", request.KeyId))

	if err != nil {
		return GetKMSKeySigningAlgorithmsResponse{}, errors.NewGenericRequestError(operationGetKMSKeySigningAlgorithms, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return GetKMSKeySigningAlgorithmsResponse{}, ErrNotFound
	}

	if response.IsError() {
		return GetKMSKeySigningAlgorithmsResponse{}, errors.NewAPIErrorWithResponse(operationGetKMSKeySigningAlgorithms, response, nil)
	}

	return signingAlgorithmsResponse, nil
}