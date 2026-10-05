package infisicalclient

import (
	"fmt"
	"maps"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type SecretValidationRuleType string

const (
	SecretValidationRuleTypeStaticSecrets   SecretValidationRuleType = "static-secrets"
	SecretValidationRuleTypeDynamicSecrets  SecretValidationRuleType = "dynamic-secrets"
	SecretValidationRuleTypeSecretRotations SecretValidationRuleType = "secret-rotations"
)

const (
	operationCreateSecretValidationRule   = "CallCreateSecretValidationRule"
	operationGetSecretValidationRuleById  = "CallGetSecretValidationRuleById"
	operationUpdateSecretValidationRule   = "CallUpdateSecretValidationRule"
	operationDeleteSecretValidationRule   = "CallDeleteSecretValidationRule"
	operationListSecretValidationRules    = "CallListSecretValidationRules"
	operationListAllSecretValidationRules = "CallListAllSecretValidationRules"
)

// secretValidationRulePath is the collection every secret validation rule endpoint hangs off.
const secretValidationRulePath = "api/v1/secret-validation-rules"

// body renders the create payload. The API expects the type-specific constraint keys beside the
// shared fields rather than nested under a wrapper, and every endpoint declares
// additionalProperties:false, so the body is assembled as a map and the caller's constraints are
// merged into it. Anything the caller leaves out of Constraints is left out of the body.
func (request CreateSecretValidationRuleRequest) body() map[string]any {
	body := map[string]any{
		"name":       request.Name,
		"projectId":  request.ProjectID,
		"secretPath": request.SecretPath,
		"isActive":   request.IsActive,
	}

	if request.Description != nil {
		body["description"] = *request.Description
	}

	if request.Environment != nil {
		body["environment"] = *request.Environment
	}

	maps.Copy(body, request.Constraints)

	return body
}

func (request UpdateSecretValidationRuleRequest) body() map[string]any {
	body := map[string]any{
		"name":        request.Name,
		"secretPath":  request.SecretPath,
		"isActive":    request.IsActive,
		"description": request.Description,
		"environment": request.Environment,
	}

	maps.Copy(body, request.Constraints)

	return body
}

func (client Client) CreateSecretValidationRule(request CreateSecretValidationRuleRequest) (SecretValidationRule, error) {
	var body CreateSecretValidationRuleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request.body()).
		Post(fmt.Sprintf("%s/%s", secretValidationRulePath, string(request.Type)))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationCreateSecretValidationRule, err)
	}

	if response.IsError() {
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationCreateSecretValidationRule, response, nil)
	}

	return body.SecretValidationRule, nil
}

func (client Client) GetSecretValidationRuleById(request GetSecretValidationRuleByIdRequest) (SecretValidationRule, error) {
	var body GetSecretValidationRuleByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("%s/%s/%s", secretValidationRulePath, string(request.Type), request.ID))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationGetSecretValidationRuleById, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return SecretValidationRule{}, ErrNotFound
	}

	if response.IsError() {
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationGetSecretValidationRuleById, response, nil)
	}

	return body.SecretValidationRule, nil
}

func (client Client) UpdateSecretValidationRule(request UpdateSecretValidationRuleRequest) (SecretValidationRule, error) {
	var body UpdateSecretValidationRuleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request.body()).
		Patch(fmt.Sprintf("%s/%s/%s", secretValidationRulePath, string(request.Type), request.ID))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationUpdateSecretValidationRule, err)
	}

	if response.IsError() {
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationUpdateSecretValidationRule, response, nil)
	}

	return body.SecretValidationRule, nil
}

func (client Client) DeleteSecretValidationRule(request DeleteSecretValidationRuleRequest) (SecretValidationRule, error) {
	var body DeleteSecretValidationRuleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s/%s", secretValidationRulePath, string(request.Type), request.ID))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationDeleteSecretValidationRule, err)
	}

	// A rule already gone is the outcome a delete wanted, so it is not an error to report.
	if response.StatusCode() == http.StatusNotFound {
		return SecretValidationRule{}, ErrNotFound
	}

	if response.IsError() {
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationDeleteSecretValidationRule, response, nil)
	}

	return body.SecretValidationRule, nil
}

// ListSecretValidationRules returns the rules of a single type in a project.
func (client Client) ListSecretValidationRules(request ListSecretValidationRulesRequest) ([]SecretValidationRule, error) {
	var body ListSecretValidationRulesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectID).
		Get(fmt.Sprintf("%s/%s", secretValidationRulePath, string(request.Type)))

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListSecretValidationRules, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationListSecretValidationRules, response, nil)
	}

	return body.SecretValidationRules, nil
}

// ListAllSecretValidationRules returns every rule in a project, of any type. Each rule carries the
// Type field, which is how callers tell the shapes apart.
func (client Client) ListAllSecretValidationRules(request ListAllSecretValidationRulesRequest) ([]SecretValidationRule, error) {
	var body ListSecretValidationRulesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectID).
		Get(secretValidationRulePath)

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListAllSecretValidationRules, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationListAllSecretValidationRules, response, nil)
	}

	return body.SecretValidationRules, nil
}
