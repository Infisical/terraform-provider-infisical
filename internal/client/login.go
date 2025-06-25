package infisicalclient

import (
	"fmt"
	"os"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationUniversalMachineIdentityAuth  = "CallUniversalMachineIdentityAuth"
	operationGetServiceTokenDetailsV2      = "CallGetServiceTokenDetailsV2"
	operationOidcMachineIdentityAuth       = "CallOidcMachineIdentityAuth"
	operationKubernetesMachineIdentityAuth = "CallKubernetesMachineIdentityAuth"
	operationTokenMachineIdentityAuth      = "CallTokenMachineIdentityAuth"
)

func (client Client) UniversalMachineIdentityAuth() (string, error) {
	if client.Config.ClientId == "" || client.Config.ClientSecret == "" {
		return "", fmt.Errorf("you must set the client secret and client ID for the client before making calls")
	}

	var loginResponse MachineIdentityAuthResponse

	res, err := client.Config.HttpClient.R().SetResult(&loginResponse).SetHeader("User-Agent", USER_AGENT).SetBody(map[string]string{
		"clientId":     client.Config.ClientId,
		"clientSecret": client.Config.ClientSecret,
	}).Post("api/v1/auth/universal-auth/login")

	if err != nil {
		return "", errors.NewGenericRequestError(operationUniversalMachineIdentityAuth, err)
	}

	if res.IsError() {
		return "", errors.NewAPIErrorWithResponse(operationUniversalMachineIdentityAuth, res, nil)
	}

	return loginResponse.AccessToken, nil
}

func (client Client) GetServiceTokenDetailsV2() (GetServiceTokenDetailsResponse, error) {
	var tokenDetailsResponse GetServiceTokenDetailsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&tokenDetailsResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v2/service-token")

	if err != nil {
		return GetServiceTokenDetailsResponse{}, errors.NewGenericRequestError(operationGetServiceTokenDetailsV2, err)
	}

	if response.IsError() {
		return GetServiceTokenDetailsResponse{}, errors.NewAPIErrorWithResponse(operationGetServiceTokenDetailsV2, response, nil)
	}

	return tokenDetailsResponse, nil
}

func (client Client) OidcMachineIdentityAuth() (string, error) {
	tokenEnvironmentName := client.Config.OidcTokenEnvName
	if tokenEnvironmentName == "" {
		tokenEnvironmentName = INFISICAL_AUTH_JWT_NAME
	}

	authJwt := os.Getenv(tokenEnvironmentName)

	if client.Config.IdentityId == "" {
		return "", fmt.Errorf("you must set the identity ID for the client before making calls")
	}

	if authJwt == "" {
		return "", fmt.Errorf("%s is not present in the environment", tokenEnvironmentName)
	}

	var loginResponse MachineIdentityAuthResponse

	res, err := client.Config.HttpClient.R().SetResult(&loginResponse).SetHeader("User-Agent", USER_AGENT).SetBody(map[string]string{
		"identityId": client.Config.IdentityId,
		"jwt":        authJwt,
	}).Post("api/v1/auth/oidc-auth/login")

	if err != nil {
		return "", errors.NewGenericRequestError(operationOidcMachineIdentityAuth, err)
	}

	if res.IsError() {
		return "", errors.NewAPIErrorWithResponse(operationOidcMachineIdentityAuth, res, nil)
	}

	return loginResponse.AccessToken, nil
}

func (client Client) KubernetesMachineIdentityAuth() (string, error) {

	token := client.Config.ServiceAccountToken
	tokenPath := client.Config.ServiceAccountTokenPath

	if tokenPath == "" {
		tokenPath = INFISICAL_KUBERNETES_SERVICE_ACCOUNT_DEFAULT_TOKEN_PATH
	}

	if token == "" {
		tokenBytes, err := os.ReadFile(tokenPath)
		if err != nil {
			return "", errors.NewGenericRequestError(operationKubernetesMachineIdentityAuth, err)
		}

		token = string(tokenBytes)
	}

	if client.Config.IdentityId == "" {
		return "", fmt.Errorf("you must set the identity ID for the client before making calls")
	}

	var loginResponse MachineIdentityAuthResponse

	res, err := client.Config.HttpClient.R().SetResult(&loginResponse).SetHeader("User-Agent", USER_AGENT).SetBody(map[string]string{
		"identityId": client.Config.IdentityId,
		"jwt":        token,
	}).Post("api/v1/auth/kubernetes-auth/login")

	if err != nil {
		return "", errors.NewGenericRequestError(operationKubernetesMachineIdentityAuth, err)
	}

	if res.IsError() {
		return "", errors.NewAPIErrorWithResponse(operationKubernetesMachineIdentityAuth, res, nil)
	}

	return loginResponse.AccessToken, nil
}

func (client Client) TokenMachineIdentityAuth() (string, error) {
	if client.Config.Token == "" {
		return "", fmt.Errorf("you must set the token for the client before making calls")
	}

	return client.Config.Token, nil
}
