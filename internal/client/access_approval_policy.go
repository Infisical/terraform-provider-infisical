package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) CreateAccessApprovalPolicy(request CreateAccessApprovalPolicyRequest) (CreateAccessApprovalPolicyResponse, error) {
	var body CreateAccessApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/access-approvals/policies")

	if err != nil {
		return CreateAccessApprovalPolicyResponse{}, fmt.Errorf("CallCreateAccessApprovalPolicy: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateAccessApprovalPolicyResponse{}, fmt.Errorf("CallCreateAccessApprovalPolicy: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) GetAccessApprovalPolicyByID(request GetAccessApprovalPolicyByIDRequest) (GetAccessApprovalPolicyByIDResponse, error) {
	var body GetAccessApprovalPolicyByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get(fmt.Sprintf("api/v1/access-approvals/policies/%s", request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return GetAccessApprovalPolicyByIDResponse{}, ErrNotFound
	}

	if err != nil {
		return GetAccessApprovalPolicyByIDResponse{}, fmt.Errorf("GetAccessApprovalPolicyById: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetAccessApprovalPolicyByIDResponse{}, fmt.Errorf("GetAccessApprovalPolicyById: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body, nil
}

func (client Client) UpdateAccessApprovalPolicy(request UpdateAccessApprovalPolicyRequest) (UpdateAccessApprovalPolicyResponse, error) {
	var body UpdateAccessApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/access-approvals/policies/%s", request.ID))

	if err != nil {
		return UpdateAccessApprovalPolicyResponse{}, fmt.Errorf("CallUpdateAccessApprovalPolicy: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateAccessApprovalPolicyResponse{}, fmt.Errorf("CallUpdateAccessApprovalPolicy: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) DeleteAccessApprovalPolicy(request DeleteAccessApprovalPolicyRequest) (DeleteAccessApprovalPolicyRequest, error) {
	var responseData DeleteAccessApprovalPolicyRequest
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v1/access-approvals/policies/%s", request.ID))

	if err != nil {
		return DeleteAccessApprovalPolicyRequest{}, fmt.Errorf("CallDeleteAccessApprovalPolicy: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteAccessApprovalPolicyRequest{}, fmt.Errorf("CallDeleteAccessApprovalPolicy: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}
