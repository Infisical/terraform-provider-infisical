package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePamAccessApprovalPolicy = "CallCreatePamAccessApprovalPolicy"
	operationGetPamAccessApprovalPolicy    = "CallGetPamAccessApprovalPolicy"
	operationUpdatePamAccessApprovalPolicy = "CallUpdatePamAccessApprovalPolicy"
	operationDeletePamAccessApprovalPolicy = "CallDeletePamAccessApprovalPolicy"
)

const pamAccessApprovalPoliciesPath = "api/v1/approval-policies/pam-access"

func (client Client) CreatePamAccessApprovalPolicy(request CreatePamAccessApprovalPolicyRequest) (PamAccessApprovalPolicy, error) {
	// PAM policies take no conditions, but the API requires the list.
	payload := struct {
		CreatePamAccessApprovalPolicyRequest
		Conditions []any `json:"conditions"`
	}{request, []any{}}

	var body PamAccessApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(payload).
		Post(pamAccessApprovalPoliciesPath)

	if err != nil {
		return PamAccessApprovalPolicy{}, errors.NewGenericRequestError(operationCreatePamAccessApprovalPolicy, err)
	}

	if response.IsError() {
		return PamAccessApprovalPolicy{}, errors.NewAPIErrorWithResponse(operationCreatePamAccessApprovalPolicy, response, nil)
	}

	return body.Policy, nil
}

func (client Client) GetPamAccessApprovalPolicy(request GetPamAccessApprovalPolicyRequest) (PamAccessApprovalPolicy, error) {
	var body PamAccessApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("%s/%s", pamAccessApprovalPoliciesPath, request.ID))

	if err != nil {
		return PamAccessApprovalPolicy{}, errors.NewGenericRequestError(operationGetPamAccessApprovalPolicy, err)
	}

	if response.IsError() {
		if isApprovalPolicyNotFound(response) {
			return PamAccessApprovalPolicy{}, ErrNotFound
		}
		return PamAccessApprovalPolicy{}, errors.NewAPIErrorWithResponse(operationGetPamAccessApprovalPolicy, response, nil)
	}

	return body.Policy, nil
}

func (client Client) UpdatePamAccessApprovalPolicy(request UpdatePamAccessApprovalPolicyRequest) (PamAccessApprovalPolicy, error) {
	var body PamAccessApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("%s/%s", pamAccessApprovalPoliciesPath, request.ID))

	if err != nil {
		return PamAccessApprovalPolicy{}, errors.NewGenericRequestError(operationUpdatePamAccessApprovalPolicy, err)
	}

	if response.IsError() {
		if isApprovalPolicyNotFound(response) {
			return PamAccessApprovalPolicy{}, ErrNotFound
		}
		return PamAccessApprovalPolicy{}, errors.NewAPIErrorWithResponse(operationUpdatePamAccessApprovalPolicy, response, nil)
	}

	return body.Policy, nil
}

func (client Client) DeletePamAccessApprovalPolicy(request DeletePamAccessApprovalPolicyRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s", pamAccessApprovalPoliciesPath, request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeletePamAccessApprovalPolicy, err)
	}

	if response.IsError() {
		if isApprovalPolicyNotFound(response) {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeletePamAccessApprovalPolicy, response, nil)
	}

	return nil
}
