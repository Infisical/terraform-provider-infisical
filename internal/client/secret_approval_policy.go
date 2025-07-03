package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateSecretApprovalPolicy  = "CallCreateSecretApprovalPolicy"
	operationGetSecretApprovalPolicyByID = "CallGetSecretApprovalPolicyByID"
	operationUpdateSecretApprovalPolicy  = "CallUpdateSecretApprovalPolicy"
	operationDeleteSecretApprovalPolicy  = "CallDeleteSecretApprovalPolicy"
)

func (client Client) CreateSecretApprovalPolicy(request CreateSecretApprovalPolicyRequest) (CreateSecretApprovalPolicyResponse, error) {
	var body CreateSecretApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/secret-approvals")

	if err != nil {
		return CreateSecretApprovalPolicyResponse{}, errors.NewGenericRequestError(operationCreateSecretApprovalPolicy, err)
	}

	if response.IsError() {
		return CreateSecretApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationCreateSecretApprovalPolicy, response, nil)
	}

	return body, nil
}

func (client Client) GetSecretApprovalPolicyByID(request GetSecretApprovalPolicyByIDRequest) (GetSecretApprovalPolicyByIDResponse, error) {
	var body GetSecretApprovalPolicyByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get(fmt.Sprintf("api/v1/secret-approvals/%s", request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return GetSecretApprovalPolicyByIDResponse{}, ErrNotFound
	}

	if err != nil {
		return GetSecretApprovalPolicyByIDResponse{}, errors.NewGenericRequestError(operationGetSecretApprovalPolicyByID, err)
	}

	if response.IsError() {
		return GetSecretApprovalPolicyByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretApprovalPolicyByID, response, nil)
	}

	return body, nil
}

func (client Client) UpdateSecretApprovalPolicy(request UpdateSecretApprovalPolicyRequest) (UpdateSecretApprovalPolicyResponse, error) {
	var body UpdateSecretApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/secret-approvals/%s", request.ID))

	if err != nil {
		return UpdateSecretApprovalPolicyResponse{}, errors.NewGenericRequestError(operationUpdateSecretApprovalPolicy, err)
	}

	if response.IsError() {
		return UpdateSecretApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationUpdateSecretApprovalPolicy, response, nil)
	}

	return body, nil
}

func (client Client) DeleteSecretApprovalPolicy(request DeleteSecretApprovalPolicyRequest) (DeleteSecretApprovalPolicyResponse, error) {
	var responseData DeleteSecretApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v1/secret-approvals/%s", request.ID))

	if err != nil {
		return DeleteSecretApprovalPolicyResponse{}, errors.NewGenericRequestError(operationDeleteSecretApprovalPolicy, err)
	}

	if response.IsError() {
		return DeleteSecretApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationDeleteSecretApprovalPolicy, response, nil)
	}

	return responseData, nil
}
