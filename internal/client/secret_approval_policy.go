package infisicalclient

import (
	"fmt"
	"net/http"
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
		return CreateSecretApprovalPolicyResponse{}, fmt.Errorf("CallCreateSecretApprovalPolicy: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateSecretApprovalPolicyResponse{}, fmt.Errorf("CallCreateSecretApprovalPolicy: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return GetSecretApprovalPolicyByIDResponse{}, fmt.Errorf("GetSecretApprovalPolicyByID: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetSecretApprovalPolicyByIDResponse{}, fmt.Errorf("GetSecretApprovalPolicyByID: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return UpdateSecretApprovalPolicyResponse{}, fmt.Errorf("CallUpdateSecretApprovalPolicy: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateSecretApprovalPolicyResponse{}, fmt.Errorf("CallUpdateSecretApprovalPolicy: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) DeleteSecretApprovalPolicy(request DeleteSecretApprovalPolicyRequest) (DeleteSecretApprovalPolicyRequest, error) {
	var responseData DeleteSecretApprovalPolicyRequest
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v1/secret-approvals/%s", request.ID))

	if err != nil {
		return DeleteSecretApprovalPolicyRequest{}, fmt.Errorf("CallDeleteSecretApprovalPolicy: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteSecretApprovalPolicyRequest{}, fmt.Errorf("CallDeleteSecretApprovalPolicy: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}
