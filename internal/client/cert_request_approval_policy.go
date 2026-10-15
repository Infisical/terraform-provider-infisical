package infisicalclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"

	"github.com/go-resty/resty/v2"
)

const (
	operationCreateCertRequestApprovalPolicy = "CallCreateCertRequestApprovalPolicy"
	operationGetCertRequestApprovalPolicy    = "CallGetCertRequestApprovalPolicy"
	operationUpdateCertRequestApprovalPolicy = "CallUpdateCertRequestApprovalPolicy"
	operationDeleteCertRequestApprovalPolicy = "CallDeleteCertRequestApprovalPolicy"
)

const certRequestApprovalPoliciesPath = "api/v1/approval-policies/cert-request"

// Infisical answers a missing policy with a 403 "Policy not found" instead of a 404.
const approvalPolicyNotFoundMessage = "Policy not found"

func isApprovalPolicyNotFound(response *resty.Response) bool {
	switch response.StatusCode() {
	case http.StatusNotFound:
		return true
	case http.StatusForbidden:
		var body struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(response.Body(), &body); err != nil {
			return false
		}
		return body.Message == approvalPolicyNotFoundMessage
	default:
		return false
	}
}

func (client Client) CreateCertRequestApprovalPolicy(request CreateCertRequestApprovalPolicyRequest) (CertRequestApprovalPolicyResponse, error) {
	var body CertRequestApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(certRequestApprovalPoliciesPath)

	if err != nil {
		return CertRequestApprovalPolicyResponse{}, errors.NewGenericRequestError(operationCreateCertRequestApprovalPolicy, err)
	}

	if response.IsError() {
		return CertRequestApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationCreateCertRequestApprovalPolicy, response, nil)
	}

	return body, nil
}

func (client Client) GetCertRequestApprovalPolicy(request GetCertRequestApprovalPolicyRequest) (CertRequestApprovalPolicyResponse, error) {
	var body CertRequestApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("%s/%s", certRequestApprovalPoliciesPath, request.ID))

	if err != nil {
		return CertRequestApprovalPolicyResponse{}, errors.NewGenericRequestError(operationGetCertRequestApprovalPolicy, err)
	}

	if response.IsError() {
		if isApprovalPolicyNotFound(response) {
			return CertRequestApprovalPolicyResponse{}, ErrNotFound
		}
		return CertRequestApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationGetCertRequestApprovalPolicy, response, nil)
	}

	return body, nil
}

func (client Client) UpdateCertRequestApprovalPolicy(request UpdateCertRequestApprovalPolicyRequest) (CertRequestApprovalPolicyResponse, error) {
	var body CertRequestApprovalPolicyResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("%s/%s", certRequestApprovalPoliciesPath, request.ID))

	if err != nil {
		return CertRequestApprovalPolicyResponse{}, errors.NewGenericRequestError(operationUpdateCertRequestApprovalPolicy, err)
	}

	if response.IsError() {
		if isApprovalPolicyNotFound(response) {
			return CertRequestApprovalPolicyResponse{}, ErrNotFound
		}
		return CertRequestApprovalPolicyResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertRequestApprovalPolicy, response, nil)
	}

	return body, nil
}

func (client Client) DeleteCertRequestApprovalPolicy(request DeleteCertRequestApprovalPolicyRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s", certRequestApprovalPoliciesPath, request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteCertRequestApprovalPolicy, err)
	}

	if response.IsError() {
		if isApprovalPolicyNotFound(response) {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeleteCertRequestApprovalPolicy, response, nil)
	}

	return nil
}
