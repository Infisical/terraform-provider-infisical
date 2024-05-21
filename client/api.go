package infisicalclient

import (
	"fmt"
)

const USER_AGENT = "terraform"

func (client Client) UniversalMachineIdentityAuth() (string, error) {
	if client.Config.ClientId == "" || client.Config.ClientSecret == "" {
		return "", fmt.Errorf("you must set the client secret and client ID for the client before making calls")
	}

	var loginResponse UniversalMachineIdentityAuthResponse

	res, err := client.Config.HttpClient.R().SetResult(&loginResponse).SetHeader("User-Agent", USER_AGENT).SetBody(map[string]string{
		"clientId":     client.Config.ClientId,
		"clientSecret": client.Config.ClientSecret,
	}).Post("api/v1/auth/universal-auth/login")

	if err != nil {
		return "", fmt.Errorf("UniversalMachineIdentityAuth: Unable to complete api request [err=%s]", err)
	}

	if res.IsError() {
		return "", fmt.Errorf("UniversalMachineIdentityAuth: Unsuccessful response: [response=%s]", res)
	}

	return loginResponse.AccessToken, nil
}

func (client Client) CallGetServiceTokenDetailsV2() (GetServiceTokenDetailsResponse, error) {
	var tokenDetailsResponse GetServiceTokenDetailsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&tokenDetailsResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v2/service-token")

	if err != nil {
		return GetServiceTokenDetailsResponse{}, fmt.Errorf("CallGetServiceTokenDetails: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetServiceTokenDetailsResponse{}, fmt.Errorf("CallGetServiceTokenDetails: Unsuccessful response: [response=%s]", response)
	}

	return tokenDetailsResponse, nil
}

func (client Client) CallGetSecretsV3(request GetEncryptedSecretsV3Request) (GetEncryptedSecretsV3Response, error) {
	var secretsResponse GetEncryptedSecretsV3Response

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("workspaceId", request.WorkspaceId)

	if request.SecretPath != "" {
		httpRequest.SetQueryParam("secretPath", request.SecretPath)
	}

	response, err := httpRequest.Get("api/v3/secrets")

	if err != nil {
		return GetEncryptedSecretsV3Response{}, fmt.Errorf("CallGetSecretsV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetEncryptedSecretsV3Response{}, fmt.Errorf("CallGetSecretsV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%v]", response.RawResponse)
	}

	return secretsResponse, nil
}

func (client Client) CallCreateSecretsV3(request CreateSecretV3Request) error {
	var secretsResponse EncryptedSecretV3
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return fmt.Errorf("CallCreateSecretsV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallCreateSecretsV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return nil
}

func (client Client) CallDeleteSecretsV3(request DeleteSecretV3Request) error {
	var secretsResponse GetEncryptedSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return fmt.Errorf("CallDeleteSecretsV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallDeleteSecretsV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return nil
}

func (client Client) CallUpdateSecretsV3(request UpdateSecretByNameV3Request) error {

	var secretsResponse GetEncryptedSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return fmt.Errorf("CallUpdateSecretsV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallUpdateSecretsV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return nil
}

func (client Client) CallGetSingleSecretByNameV3(request GetSingleSecretByNameV3Request) (GetSingleSecretByNameSecretResponse, error) {
	var secretsResponse GetSingleSecretByNameSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.WorkspaceId).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("type", request.Type).
		SetQueryParam("secretPath", request.SecretPath).
		Get(fmt.Sprintf("api/v3/secrets/%s", request.SecretName))

	if err != nil {
		return GetSingleSecretByNameSecretResponse{}, fmt.Errorf("CallGetSingleSecretByNameV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetSingleSecretByNameSecretResponse{}, fmt.Errorf("CallGetSingleSecretByNameV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return secretsResponse, nil
}

func (client Client) CallGetSecretsRawV3(request GetRawSecretsV3Request) (GetRawSecretsV3Response, error) {
	var secretsResponse GetRawSecretsV3Response

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("workspaceId", request.WorkspaceId)

	if request.SecretPath != "" {
		httpRequest.SetQueryParam("secretPath", request.SecretPath)
	}

	response, err := httpRequest.Get("api/v3/secrets/raw")

	if err != nil {
		return GetRawSecretsV3Response{}, fmt.Errorf("CallGetSecretsRawV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetRawSecretsV3Response{}, fmt.Errorf("CallGetSecretsRawV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%v]", response.RawResponse)
	}

	return secretsResponse, nil
}

func (client Client) CallCreateRawSecretsV3(request CreateRawSecretV3Request) error {
	var secretsResponse EncryptedSecretV3
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretKey))

	if err != nil {
		return fmt.Errorf("CallCreateRawSecretsV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallCreateRawSecretsV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return nil
}

func (client Client) CallDeleteRawSecretV3(request DeleteRawSecretV3Request) error {
	var secretsResponse GetRawSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretName))

	if err != nil {
		return fmt.Errorf("CallDeleteRawSecretV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallDeleteRawSecretV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return nil
}

func (client Client) CallUpdateRawSecretV3(request UpdateRawSecretByNameV3Request) error {
	var secretsResponse GetRawSecretsV3Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretName))

	if err != nil {
		return fmt.Errorf("CallUpdateRawSecretV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallUpdateRawSecretV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return nil
}

func (client Client) CallGetSingleRawSecretByNameV3(request GetSingleSecretByNameV3Request) (GetSingleRawSecretByNameSecretResponse, error) {
	var secretsResponse GetSingleRawSecretByNameSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.WorkspaceId).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("type", request.Type).
		SetQueryParam("secretPath", request.SecretPath).
		Get(fmt.Sprintf("api/v3/secrets/raw/%s", request.SecretName))

	if err != nil {
		return GetSingleRawSecretByNameSecretResponse{}, fmt.Errorf("CallGetSingleRawSecretByNameV3: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetSingleRawSecretByNameSecretResponse{}, fmt.Errorf("CallGetSingleRawSecretByNameV3: Unsuccessful response. Please make sure your secret path, workspace and environment name are all correct [response=%s]", response)
	}

	return secretsResponse, nil
}

func (client Client) CallCreateProject(request CreateProjectRequest) (CreateProjectResponse, error) {

	if request.Slug == "" {
		request = CreateProjectRequest{
			ProjectName:      request.ProjectName,
			OrganizationSlug: request.OrganizationSlug,
		}
	}

	var projectResponse CreateProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v2/workspace")

	if err != nil {
		return CreateProjectResponse{}, fmt.Errorf("CallCreateProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectResponse{}, fmt.Errorf("CallCreateProject: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) CallDeleteProject(request DeleteProjectRequest) error {
	var projectResponse DeleteProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v2/workspace/%s", request.Slug))

	if err != nil {
		return fmt.Errorf("CallDeleteProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallDeleteProject: Unsuccessful response. [response=%s]", response)
	}

	return nil
}

func (client Client) CallGetProject(request GetProjectRequest) (ProjectWithEnvironments, error) {
	var projectResponse ProjectWithEnvironments
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v2/workspace/%s", request.Slug))

	if err != nil {
		return ProjectWithEnvironments{}, fmt.Errorf("CallGetProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return ProjectWithEnvironments{}, fmt.Errorf("CallGetProject: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) CallUpdateProject(request UpdateProjectRequest) (UpdateProjectResponse, error) {
	var projectResponse UpdateProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v2/workspace/%s", request.Slug))

	if err != nil {
		return UpdateProjectResponse{}, fmt.Errorf("CallUpdateProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectResponse{}, fmt.Errorf("CallUpdateProject: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) CallInviteUsersToProject(request InviteUsersToProjectRequest) ([]ProjectMemberships, error) {
	var inviteUsersToProjectResponse InviteUsersToProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&inviteUsersToProjectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/memberships", request.ProjectID))

	if err != nil {
		return nil, fmt.Errorf("CallInviteUsersToProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return nil, fmt.Errorf("InviteUsersToProjectRequest: Unsuccessful response. [response=%s]", response)
	}

	return inviteUsersToProjectResponse.Members, nil
}

func (client Client) CallDeleteProjectUser(request DeleteProjectUserRequest) (DeleteProjectUserResponse, error) {
	var projectUserResponse DeleteProjectUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectUserResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v2/workspace/%s/memberships", request.ProjectID))

	if err != nil {
		return DeleteProjectUserResponse{}, fmt.Errorf("CallDeleteProjectUser: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectUserResponse{}, fmt.Errorf("CallDeleteProjectUser: Unsuccessful response. [response=%s]", response)
	}

	return projectUserResponse, nil
}

func (client Client) CallUpdateProjectUser(request UpdateProjectUserRequest) (UpdateProjectUserResponse, error) {
	var projectUserResponse UpdateProjectUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectUserResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/workspace/%s/memberships/%s", request.ProjectID, request.MembershipID))

	if err != nil {
		return UpdateProjectUserResponse{}, fmt.Errorf("UpdateProjectUserResponse: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectUserResponse{}, fmt.Errorf("UpdateProjectUserResponse: Unsuccessful response. [response=%s]", response)
	}

	return projectUserResponse, nil
}

func (client Client) CallGetProjectUserByUsername(request GetProjectUserByUserNameRequest) (GetProjectUserByUserNameResponse, error) {
	var projectUserResponse GetProjectUserByUserNameResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectUserResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/workspace/%s/memberships/details", request.ProjectID))

	if err != nil {
		return GetProjectUserByUserNameResponse{}, fmt.Errorf("CallCreateProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectUserByUserNameResponse{}, fmt.Errorf("CallCreateProject: Unsuccessful response. [response=%s]", response)
	}

	return projectUserResponse, nil
}
