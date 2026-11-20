package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetSecretImport                  = "CallGetSecretImport"
	operationGetSecretImportWithSourceProject = "CallGetSecretImportWithSourceProject"
	operationGetSecretImportByID              = "CallGetSecretImportByID"
	operationGetSecretImportList              = "CallGetSecretImportList"
	operationCreateSecretImport               = "CallCreateSecretImport"
	operationUpdateSecretImport               = "CallUpdateSecretImport"
	operationDeleteSecretImport               = "CallDeleteSecretImport"
)

// The 403 messages that createImport in Infisical core (secret-import-service.ts) returns for a
// cross-project import. Both are 403 ForbiddenError, like a missing permission, so only the message
// tells them apart.
const (
	crossProjectSharingDisabledMessage = "Cross-project secret sharing is not enabled for this organization"
	crossProjectGrantMissingMessage    = "No project grant found allowing this cross-project import"
)

// CreateSecretImportRequest with the v2 field names: v2 names the destination project projectId,
// where v1 names it workspaceId.
type createSecretImportV2Request struct {
	ProjectID     string             `json:"projectId"`
	Environment   string             `json:"environment"`
	SecretPath    string             `json:"path"`
	IsReplication bool               `json:"isReplication"`
	ImportFrom    SecretImportSource `json:"import"`
}

// Workaround to getSecretImportById API call.
func findSecretImportByID(secretImports []SecretImport, id string) (GetSecretImportResponse, error) {
	for _, secretImport := range secretImports {
		if secretImport.ID == id {
			return GetSecretImportResponse{SecretImport: secretImport}, nil
		}
	}

	return GetSecretImportResponse{}, NewNotFoundError("SecretImport", id)
}

func (client Client) GetSecretImport(request GetSecretImportRequest) (GetSecretImportResponse, error) {
	var body ListSecretImportResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.ProjectID).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("path", request.SecretPath)

	response, err := httpRequest.Get("api/v1/secret-imports")

	if err != nil {
		return GetSecretImportResponse{}, errors.NewGenericRequestError(operationGetSecretImport, err)
	}

	if response.IsError() {
		return GetSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretImport, response, nil)
	}

	return findSecretImportByID(body.SecretImports, request.ID)

}

// GetSecretImportWithSourceProject reads an import through the v2 list route, the only route that
// returns its source project (importEnv.projectId). The v1 routes, and the v2 create and get by ID
// routes, leave it out. When the organization turns cross-project secret sharing off, the list leaves
// out cross-project imports, so they read as not found.
func (client Client) GetSecretImportWithSourceProject(request GetSecretImportRequest) (GetSecretImportResponse, error) {
	var body ListSecretImportResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectId", request.ProjectID).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("path", request.SecretPath).
		Get("api/v2/secret-imports")

	if err != nil {
		return GetSecretImportResponse{}, errors.NewGenericRequestError(operationGetSecretImportWithSourceProject, err)
	}

	if response.IsError() {
		return GetSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretImportWithSourceProject, response, nil)
	}

	return findSecretImportByID(body.SecretImports, request.ID)
}

func (client Client) GetSecretImportByID(request GetSecretImportByIDRequest) (GetSecretImportByIDResponse, error) {
	var body GetSecretImportByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/secret-imports/" + request.ID)

	if err != nil {
		return GetSecretImportByIDResponse{}, errors.NewGenericRequestError(operationGetSecretImportByID, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetSecretImportByIDResponse{}, ErrNotFound
		}
		return GetSecretImportByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretImportByID, response, nil)
	}

	return body, nil
}

func (client Client) GetSecretImportList(request ListSecretImportRequest) (ListSecretImportResponse, error) {
	var body ListSecretImportResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.ProjectID).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("path", request.SecretPath)

	response, err := httpRequest.Get("api/v1/secret-imports")

	if err != nil {
		return ListSecretImportResponse{}, errors.NewGenericRequestError(operationGetSecretImportList, err)
	}

	if response.IsError() {
		return ListSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretImportList, response, nil)
	}

	return body, nil
}

func (client Client) CreateSecretImport(request CreateSecretImportRequest) (CreateSecretImportResponse, error) {
	url, payload := "api/v1/secret-imports", any(request)
	if request.ImportFrom.SourceProjectID != "" {
		// v1 drops import.sourceProjectId without an error and would create a same-project import.
		url = "api/v2/secret-imports"
		payload = createSecretImportV2Request(request)
	}

	var body CreateSecretImportResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(payload).
		Post(url)

	if err != nil {
		return CreateSecretImportResponse{}, errors.NewGenericRequestError(operationCreateSecretImport, err)
	}

	if response.IsError() {
		apiError := errors.NewAPIErrorWithResponse(operationCreateSecretImport, response, nil)
		if apiErr, ok := apiError.(*errors.APIError); ok && apiErr.StatusCode == http.StatusForbidden {
			switch apiErr.ErrorMessage {
			case crossProjectSharingDisabledMessage:
				return CreateSecretImportResponse{}, fmt.Errorf("%w: %w", ErrCrossProjectSharingDisabled, apiError)
			case crossProjectGrantMissingMessage:
				return CreateSecretImportResponse{}, fmt.Errorf("%w: %w", ErrCrossProjectGrantMissing, apiError)
			}
		}
		return CreateSecretImportResponse{}, apiError
	}

	return body, nil
}

func (client Client) UpdateSecretImport(request UpdateSecretImportRequest) (UpdateSecretImportResponse, error) {
	var body UpdateSecretImportResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/secret-imports/" + request.ID)

	if err != nil {
		return UpdateSecretImportResponse{}, errors.NewGenericRequestError(operationUpdateSecretImport, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return UpdateSecretImportResponse{}, NewNotFoundError("SecretImport", request.SecretPath)
		}

		return UpdateSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationUpdateSecretImport, response, nil)
	}

	return body, nil
}

func (client Client) DeleteSecretImport(request DeleteSecretImportRequest) (DeleteSecretImportResponse, error) {
	var body DeleteSecretImportResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/secret-imports/" + request.ID)

	if err != nil {
		return DeleteSecretImportResponse{}, errors.NewGenericRequestError(operationDeleteSecretImport, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return DeleteSecretImportResponse{}, NewNotFoundError("SecretImport", request.SecretPath)
		}

		return DeleteSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationDeleteSecretImport, response, nil)
	}

	return body, nil
}
