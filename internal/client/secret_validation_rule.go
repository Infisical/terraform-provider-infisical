package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateSecretValidationRule = "CallCreateSecretValidationRule"
	operationListSecretValidationRules  = "CallListSecretValidationRules"
	operationUpdateSecretValidationRule = "CallUpdateSecretValidationRule"
	operationDeleteSecretValidationRule = "CallDeleteSecretValidationRule"
)

func (client Client) CreateSecretValidationRule(request CreateSecretValidationRuleRequest) (SecretValidationRule, error) {
	var body CreateSecretValidationRuleResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/projects/%s/secret-validation-rules", request.ProjectID))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationCreateSecretValidationRule, err)
	}

	if response.IsError() {
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationCreateSecretValidationRule, response, nil)
	}

	return body.Rule, nil
}

func (client Client) ListSecretValidationRules(request ListSecretValidationRulesRequest) (ListSecretValidationRulesResponse, error) {
	var body ListSecretValidationRulesResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/projects/%s/secret-validation-rules", request.ProjectID))

	if err != nil {
		return ListSecretValidationRulesResponse{}, errors.NewGenericRequestError(operationListSecretValidationRules, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ListSecretValidationRulesResponse{}, ErrNotFound
		}
		return ListSecretValidationRulesResponse{}, errors.NewAPIErrorWithResponse(operationListSecretValidationRules, response, nil)
	}

	return body, nil
}

// GetSecretValidationRuleById resolves a single rule by ID. The API exposes no
// GET-by-id endpoint, so this lists the project's rules and filters client-side.
// It returns ErrNotFound when no rule with the given ID exists in the project.
func (client Client) GetSecretValidationRuleById(request GetSecretValidationRuleByIdRequest) (SecretValidationRule, error) {
	rules, err := client.ListSecretValidationRules(ListSecretValidationRulesRequest{ProjectID: request.ProjectID})
	if err != nil {
		return SecretValidationRule{}, err
	}

	for _, rule := range rules.Rules {
		if rule.ID == request.RuleID {
			return rule, nil
		}
	}

	return SecretValidationRule{}, ErrNotFound
}

func (client Client) UpdateSecretValidationRule(request UpdateSecretValidationRuleRequest) (SecretValidationRule, error) {
	var body UpdateSecretValidationRuleResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/projects/%s/secret-validation-rules/%s", request.ProjectID, request.RuleID))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationUpdateSecretValidationRule, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return SecretValidationRule{}, ErrNotFound
		}
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationUpdateSecretValidationRule, response, nil)
	}

	return body.Rule, nil
}

func (client Client) DeleteSecretValidationRule(request DeleteSecretValidationRuleRequest) (SecretValidationRule, error) {
	var body DeleteSecretValidationRuleResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/projects/%s/secret-validation-rules/%s", request.ProjectID, request.RuleID))

	if err != nil {
		return SecretValidationRule{}, errors.NewGenericRequestError(operationDeleteSecretValidationRule, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return SecretValidationRule{}, ErrNotFound
		}
		return SecretValidationRule{}, errors.NewAPIErrorWithResponse(operationDeleteSecretValidationRule, response, nil)
	}

	return body.Rule, nil
}
