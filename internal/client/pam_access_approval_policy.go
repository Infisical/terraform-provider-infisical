package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePamAccessApprovalPolicy = "CallCreatePamAccessApprovalPolicy"
	operationListPamAccessApprovalPolicies = "CallListPamAccessApprovalPolicies"
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

// ListPamAccessApprovalPolicies returns the folder's approval policies, at most one since PAM allows one per folder.
func (client Client) ListPamAccessApprovalPolicies(request ListPamAccessApprovalPoliciesRequest) ([]PamAccessApprovalPolicy, error) {
	var body ListPamAccessApprovalPoliciesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParams(map[string]string{"scope": "pam-folder", "id": request.FolderID}).
		Get(pamAccessApprovalPoliciesPath)

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListPamAccessApprovalPolicies, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, errors.NewAPIErrorWithResponse(operationListPamAccessApprovalPolicies, response, nil)
	}

	return body.Policies, nil
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
