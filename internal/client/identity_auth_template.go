package infisicalclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateIdentityAuthTemplate        = "CallCreateIdentityAuthTemplate"
	operationGetIdentityAuthTemplate           = "CallGetIdentityAuthTemplate"
	operationUpdateIdentityAuthTemplate        = "CallUpdateIdentityAuthTemplate"
	operationDeleteIdentityAuthTemplate        = "CallDeleteIdentityAuthTemplate"
	operationListIdentityAuthTemplatesByMethod = "CallListIdentityAuthTemplatesByMethod"
)

const (
	IdentityAuthTemplateMethodKubernetes = "kubernetes"
	IdentityAuthTemplateMethodOidc       = "oidc"
)

// IdentityAuthTemplateMethodMismatchError is returned when a template ID resolves to a template of
// another auth method, so a kubernetes template resource cannot silently adopt an OIDC template.
type IdentityAuthTemplateMethodMismatchError struct {
	TemplateID     string
	ExpectedMethod string
	ActualMethod   string
}

func (e *IdentityAuthTemplateMethodMismatchError) Error() string {
	return fmt.Sprintf("identity auth template %s is a %s template, not a %s template", e.TemplateID, e.ActualMethod, e.ExpectedMethod)
}

// IdentityAuthTemplateAmbiguousNameError is returned by a lookup by name that matches more than one
// template. The API does not enforce unique names, so a name alone cannot always pick one.
type IdentityAuthTemplateAmbiguousNameError struct {
	Name        string
	AuthMethod  string
	TemplateIDs []string
}

func (e *IdentityAuthTemplateAmbiguousNameError) Error() string {
	return fmt.Sprintf("%d %s auth templates are named %q (IDs: %v)", len(e.TemplateIDs), e.AuthMethod, e.Name, e.TemplateIDs)
}

// The API discriminates templateFields on authMethod, so the fields are decoded only once the
// method is known.
type identityAuthTemplateRaw struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	OrgID          string          `json:"orgId"`
	AuthMethod     string          `json:"authMethod"`
	TemplateFields json.RawMessage `json:"templateFields"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

type IdentityKubernetesAuthTemplateFields struct {
	TokenReviewMode      string  `json:"tokenReviewMode"`
	KubernetesHost       *string `json:"kubernetesHost"`
	CaCert               string  `json:"caCert"`
	VerifyTlsCertificate *bool   `json:"verifyTlsCertificate"`
	HasTokenReviewerJwt  bool    `json:"hasTokenReviewerJwt"`
	GatewayID            *string `json:"gatewayId"`
	GatewayPoolID        *string `json:"gatewayPoolId"`
	AllowedAudience      string  `json:"allowedAudience"`
}

type IdentityKubernetesAuthTemplate struct {
	ID             string
	Name           string
	OrgID          string
	TemplateFields IdentityKubernetesAuthTemplateFields
}

type IdentityOidcAuthTemplateFields struct {
	OidcDiscoveryUrl string `json:"oidcDiscoveryUrl"`
	BoundIssuer      string `json:"boundIssuer"`
	BoundAudiences   string `json:"boundAudiences"`
	CaCert           string `json:"caCert"`
}

type IdentityOidcAuthTemplate struct {
	ID             string
	Name           string
	OrgID          string
	TemplateFields IdentityOidcAuthTemplateFields
}

type CreateIdentityKubernetesAuthTemplateFields struct {
	TokenReviewMode      string  `json:"tokenReviewMode"`
	KubernetesHost       *string `json:"kubernetesHost,omitempty"`
	CaCert               string  `json:"caCert,omitempty"`
	VerifyTlsCertificate *bool   `json:"verifyTlsCertificate,omitempty"`
	TokenReviewerJwt     string  `json:"tokenReviewerJwt,omitempty"`
	GatewayID            *string `json:"gatewayId,omitempty"`
	GatewayPoolID        *string `json:"gatewayPoolId,omitempty"`
	AllowedAudience      string  `json:"allowedAudience"`
}

type CreateIdentityKubernetesAuthTemplateRequest struct {
	Name           string
	TemplateFields CreateIdentityKubernetesAuthTemplateFields
}

type CreateIdentityOidcAuthTemplateFields struct {
	OidcDiscoveryUrl string `json:"oidcDiscoveryUrl"`
	BoundIssuer      string `json:"boundIssuer"`
	BoundAudiences   string `json:"boundAudiences"`
	CaCert           string `json:"caCert,omitempty"`
}

type CreateIdentityOidcAuthTemplateRequest struct {
	Name           string
	TemplateFields CreateIdentityOidcAuthTemplateFields
}

// UpdateIdentityAuthTemplateRequest is a partial update: the API merges TemplateFields into the
// stored fields, so a key left out keeps its value and a key set to nil clears it. Every change
// propagates to all identities linked to the template, so callers send only what changed.
type UpdateIdentityAuthTemplateRequest struct {
	ID             string         `json:"-"`
	Name           *string        `json:"name,omitempty"`
	TemplateFields map[string]any `json:"templateFields,omitempty"`
}

type createIdentityAuthTemplateBody struct {
	Name           string `json:"name"`
	AuthMethod     string `json:"authMethod"`
	TemplateFields any    `json:"templateFields"`
}

func (client Client) createIdentityAuthTemplate(body createIdentityAuthTemplateBody) (identityAuthTemplateRaw, error) {
	var template identityAuthTemplateRaw
	response, err := client.Config.HttpClient.
		R().
		SetResult(&template).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(body).
		Post("api/v1/identity-templates")

	if err != nil {
		return identityAuthTemplateRaw{}, errors.NewGenericRequestError(operationCreateIdentityAuthTemplate, err)
	}

	if response.IsError() {
		return identityAuthTemplateRaw{}, errors.NewAPIErrorWithResponse(operationCreateIdentityAuthTemplate, response, nil)
	}

	return template, nil
}

func (client Client) getIdentityAuthTemplate(templateID string) (identityAuthTemplateRaw, error) {
	var template identityAuthTemplateRaw
	response, err := client.Config.HttpClient.
		R().
		SetResult(&template).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/identity-templates/" + url.PathEscape(templateID))

	if err != nil {
		return identityAuthTemplateRaw{}, errors.NewGenericRequestError(operationGetIdentityAuthTemplate, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return identityAuthTemplateRaw{}, ErrNotFound
	}

	if response.IsError() {
		return identityAuthTemplateRaw{}, errors.NewAPIErrorWithResponse(operationGetIdentityAuthTemplate, response, nil)
	}

	return template, nil
}

func (client Client) updateIdentityAuthTemplate(request UpdateIdentityAuthTemplateRequest) (identityAuthTemplateRaw, error) {
	var template identityAuthTemplateRaw
	response, err := client.Config.HttpClient.
		R().
		SetResult(&template).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/identity-templates/" + url.PathEscape(request.ID))

	if err != nil {
		return identityAuthTemplateRaw{}, errors.NewGenericRequestError(operationUpdateIdentityAuthTemplate, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return identityAuthTemplateRaw{}, ErrNotFound
	}

	if response.IsError() {
		return identityAuthTemplateRaw{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityAuthTemplate, response, nil)
	}

	return template, nil
}

func (client Client) listIdentityAuthTemplatesByMethod(authMethod string) ([]identityAuthTemplateRaw, error) {
	var templates []identityAuthTemplateRaw
	response, err := client.Config.HttpClient.
		R().
		SetResult(&templates).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("authMethod", authMethod).
		Get("api/v1/identity-templates")

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListIdentityAuthTemplatesByMethod, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationListIdentityAuthTemplatesByMethod, response, nil)
	}

	return templates, nil
}

// findIdentityAuthTemplateByName returns ErrNotFound only when the list arrived without a match,
// since a list that never arrived cannot prove the template's absence.
func (client Client) findIdentityAuthTemplateByName(authMethod string, name string) (identityAuthTemplateRaw, error) {
	templates, err := client.listIdentityAuthTemplatesByMethod(authMethod)
	if err != nil {
		return identityAuthTemplateRaw{}, err
	}

	var matches []identityAuthTemplateRaw
	for _, template := range templates {
		if template.Name == name {
			matches = append(matches, template)
		}
	}

	switch len(matches) {
	case 0:
		return identityAuthTemplateRaw{}, ErrNotFound
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, len(matches))
		for i, match := range matches {
			ids[i] = match.ID
		}
		return identityAuthTemplateRaw{}, &IdentityAuthTemplateAmbiguousNameError{Name: name, AuthMethod: authMethod, TemplateIDs: ids}
	}
}

func (client Client) DeleteIdentityAuthTemplate(templateID string) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete("api/v1/identity-templates/" + url.PathEscape(templateID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteIdentityAuthTemplate, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteIdentityAuthTemplate, response, nil)
	}

	return nil
}

func decodeIdentityAuthTemplateFields(raw identityAuthTemplateRaw, expectedMethod string, target any) error {
	if raw.AuthMethod != expectedMethod {
		return &IdentityAuthTemplateMethodMismatchError{TemplateID: raw.ID, ExpectedMethod: expectedMethod, ActualMethod: raw.AuthMethod}
	}
	if err := json.Unmarshal(raw.TemplateFields, target); err != nil {
		return fmt.Errorf("could not decode the fields of %s auth template %s: %w", expectedMethod, raw.ID, err)
	}
	return nil
}

func toIdentityKubernetesAuthTemplate(raw identityAuthTemplateRaw, err error) (IdentityKubernetesAuthTemplate, error) {
	if err != nil {
		return IdentityKubernetesAuthTemplate{}, err
	}

	template := IdentityKubernetesAuthTemplate{ID: raw.ID, Name: raw.Name, OrgID: raw.OrgID}
	if err := decodeIdentityAuthTemplateFields(raw, IdentityAuthTemplateMethodKubernetes, &template.TemplateFields); err != nil {
		return IdentityKubernetesAuthTemplate{}, err
	}
	return template, nil
}

func toIdentityOidcAuthTemplate(raw identityAuthTemplateRaw, err error) (IdentityOidcAuthTemplate, error) {
	if err != nil {
		return IdentityOidcAuthTemplate{}, err
	}

	template := IdentityOidcAuthTemplate{ID: raw.ID, Name: raw.Name, OrgID: raw.OrgID}
	if err := decodeIdentityAuthTemplateFields(raw, IdentityAuthTemplateMethodOidc, &template.TemplateFields); err != nil {
		return IdentityOidcAuthTemplate{}, err
	}
	return template, nil
}

func (client Client) CreateIdentityKubernetesAuthTemplate(request CreateIdentityKubernetesAuthTemplateRequest) (IdentityKubernetesAuthTemplate, error) {
	return toIdentityKubernetesAuthTemplate(client.createIdentityAuthTemplate(createIdentityAuthTemplateBody{
		Name:           request.Name,
		AuthMethod:     IdentityAuthTemplateMethodKubernetes,
		TemplateFields: request.TemplateFields,
	}))
}

func (client Client) GetIdentityKubernetesAuthTemplate(templateID string) (IdentityKubernetesAuthTemplate, error) {
	return toIdentityKubernetesAuthTemplate(client.getIdentityAuthTemplate(templateID))
}

func (client Client) GetIdentityKubernetesAuthTemplateByName(name string) (IdentityKubernetesAuthTemplate, error) {
	return toIdentityKubernetesAuthTemplate(client.findIdentityAuthTemplateByName(IdentityAuthTemplateMethodKubernetes, name))
}

func (client Client) UpdateIdentityKubernetesAuthTemplate(request UpdateIdentityAuthTemplateRequest) (IdentityKubernetesAuthTemplate, error) {
	return toIdentityKubernetesAuthTemplate(client.updateIdentityAuthTemplate(request))
}

func (client Client) CreateIdentityOidcAuthTemplate(request CreateIdentityOidcAuthTemplateRequest) (IdentityOidcAuthTemplate, error) {
	return toIdentityOidcAuthTemplate(client.createIdentityAuthTemplate(createIdentityAuthTemplateBody{
		Name:           request.Name,
		AuthMethod:     IdentityAuthTemplateMethodOidc,
		TemplateFields: request.TemplateFields,
	}))
}

func (client Client) GetIdentityOidcAuthTemplate(templateID string) (IdentityOidcAuthTemplate, error) {
	return toIdentityOidcAuthTemplate(client.getIdentityAuthTemplate(templateID))
}

func (client Client) GetIdentityOidcAuthTemplateByName(name string) (IdentityOidcAuthTemplate, error) {
	return toIdentityOidcAuthTemplate(client.findIdentityAuthTemplateByName(IdentityAuthTemplateMethodOidc, name))
}

func (client Client) UpdateIdentityOidcAuthTemplate(request UpdateIdentityAuthTemplateRequest) (IdentityOidcAuthTemplate, error) {
	return toIdentityOidcAuthTemplate(client.updateIdentityAuthTemplate(request))
}
