package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateAuditLogStream = "CallCreateAuditLogStream"
	operationGetAuditLogStream    = "CallGetAuditLogStream"
	operationUpdateAuditLogStream = "CallUpdateAuditLogStream"
	operationDeleteAuditLogStream = "CallDeleteAuditLogStream"
)

// AuditLogStreamRedactedCredential is the sentinel the API substitutes for secrets it will not
// disclose. Sent back on update, it means "keep the stored value".
const AuditLogStreamRedactedCredential = "******"

// Audit log stream providers. Each maps to its own API route segment.
const (
	AuditLogStreamProviderAzure     = "azure"
	AuditLogStreamProviderCribl     = "cribl"
	AuditLogStreamProviderCustom    = "custom"
	AuditLogStreamProviderDatadog   = "datadog"
	AuditLogStreamProviderSplunk    = "splunk"
	AuditLogStreamProviderSumoLogic = "sumo-logic"
)

// AuditLogStreamFilters scopes which events a stream receives. An empty Products list streams
// every product.
type AuditLogStreamFilters struct {
	Products []string `json:"products,omitempty"`
}

type AuditLogStream struct {
	ID         string `json:"id"`
	OrgID      string `json:"orgId"`
	Provider   string `json:"provider"`
	StreamMode string `json:"streamMode"`
	// Credentials come back sanitized: secret values are redacted or omitted entirely.
	Credentials map[string]any         `json:"credentials"`
	Filters     *AuditLogStreamFilters `json:"filters"`
}

type AuditLogStreamResponse struct {
	AuditLogStream AuditLogStream `json:"auditLogStream"`
}

type CreateAuditLogStreamRequest struct {
	Provider    string         `json:"-"`
	Credentials map[string]any `json:"credentials"`
	// Sent even when nil so an omitted filter block clears any existing scoping.
	Filters *AuditLogStreamFilters `json:"filters"`
}

type UpdateAuditLogStreamRequest struct {
	ID          string                 `json:"-"`
	Provider    string                 `json:"-"`
	Credentials map[string]any         `json:"credentials"`
	Filters     *AuditLogStreamFilters `json:"filters"`
}

type AuditLogStreamByIdRequest struct {
	ID       string
	Provider string
}

func (client Client) CreateAuditLogStream(request CreateAuditLogStreamRequest) (AuditLogStream, error) {
	var responseData AuditLogStreamResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(auditLogStreamProviderPath(request.Provider))

	if err != nil {
		return AuditLogStream{}, errors.NewGenericRequestError(operationCreateAuditLogStream, err)
	}
	if response.IsError() {
		return AuditLogStream{}, errors.NewAPIErrorWithResponse(operationCreateAuditLogStream, response, nil)
	}
	return responseData.AuditLogStream, nil
}

func (client Client) GetAuditLogStreamById(request AuditLogStreamByIdRequest) (AuditLogStream, error) {
	var responseData AuditLogStreamResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get(auditLogStreamPath(request.Provider, request.ID))

	if err != nil {
		return AuditLogStream{}, errors.NewGenericRequestError(operationGetAuditLogStream, err)
	}
	if response.StatusCode() == http.StatusNotFound {
		return AuditLogStream{}, ErrNotFound
	}
	if response.IsError() {
		return AuditLogStream{}, errors.NewAPIErrorWithResponse(operationGetAuditLogStream, response, nil)
	}
	return responseData.AuditLogStream, nil
}

func (client Client) UpdateAuditLogStream(request UpdateAuditLogStreamRequest) (AuditLogStream, error) {
	var responseData AuditLogStreamResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(auditLogStreamPath(request.Provider, request.ID))

	if err != nil {
		return AuditLogStream{}, errors.NewGenericRequestError(operationUpdateAuditLogStream, err)
	}
	if response.StatusCode() == http.StatusNotFound {
		return AuditLogStream{}, ErrNotFound
	}
	if response.IsError() {
		return AuditLogStream{}, errors.NewAPIErrorWithResponse(operationUpdateAuditLogStream, response, nil)
	}
	return responseData.AuditLogStream, nil
}

func (client Client) DeleteAuditLogStream(request AuditLogStreamByIdRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(auditLogStreamPath(request.Provider, request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteAuditLogStream, err)
	}
	if response.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}
	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteAuditLogStream, response, nil)
	}
	return nil
}

func auditLogStreamProviderPath(provider string) string {
	return fmt.Sprintf("api/v1/audit-log-streams/%s", provider)
}

func auditLogStreamPath(provider, id string) string {
	return fmt.Sprintf("api/v1/audit-log-streams/%s/%s", provider, id)
}
