package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCallCreateAccessApprovalPolicy = "CallCreateAccessApprovalPolicy"
	operationGetAccessApprovalPolicyById    = "GetAccessApprovalPolicyById"
	operationCallUpdateAccessApprovalPolicy = "CallUpdateAccessApprovalPolicy"
	operationCallDeleteAccessApprovalPolicy = "CallDeleteAccessApprovalPolicy"
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
		return CreateAccessApprovalPolicyResponse{}, errors.NewGenericRequestError(operationCallCreateAccessApprovalPolicy, err)
	}

	if response.IsError() {
		return CreateAccessApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationCallCreateAccessApprovalPolicy, response, nil)
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
		return GetAccessApprovalPolicyByIDResponse{}, errors.NewGenericRequestError(operationGetAccessApprovalPolicyById, err)
	}

	if response.IsError() {
		return GetAccessApprovalPolicyByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetAccessApprovalPolicyById, response, nil)
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
		return UpdateAccessApprovalPolicyResponse{}, errors.NewGenericRequestError(operationCallUpdateAccessApprovalPolicy, err)
	}

	if response.IsError() {
		return UpdateAccessApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationCallUpdateAccessApprovalPolicy, response, nil)
	}

	return body, nil
}

func (client Client) DeleteAccessApprovalPolicy(request DeleteAccessApprovalPolicyRequest) (DeleteAccessApprovalPolicyResponse, error) {
	var responseData DeleteAccessApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v1/access-approvals/policies/%s", request.ID))

	if err != nil {
		return DeleteAccessApprovalPolicyResponse{}, errors.NewGenericRequestError(operationCallDeleteAccessApprovalPolicy, err)
	}

	if response.IsError() {
		return DeleteAccessApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationCallDeleteAccessApprovalPolicy, response, nil)
	}

	return responseData, nil
}
