package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentity      = "CallGetIdentity"
	operationSearchIdentities = "CallSearchIdentities"
	operationCreateIdentity   = "CallCreateIdentity"
	operationUpdateIdentity   = "CallUpdateIdentity"
	operationDeleteIdentity   = "CallDeleteIdentity"
)

// searchIdentitiesPageSize is the largest page the search endpoint accepts.
const searchIdentitiesPageSize = 100

func (client Client) GetIdentity(request GetIdentityRequest) (OrgIdentity, error) {
	var body GetIdentityResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return OrgIdentity{}, ErrNotFound
	}

	if err != nil {
		return OrgIdentity{}, errors.NewGenericRequestError(operationGetIdentity, err)
	}

	if response.IsError() {
		return OrgIdentity{}, errors.NewAPIErrorWithResponse(operationGetIdentity, response, nil)
	}

	return body.Identity, nil
}

// SearchIdentitiesByName returns the identities in the caller's organization whose
// name is exactly name. The API has no get-by-name endpoint, so search is the only
// name-based lookup, and it returns a collection because identity names are not
// unique. Deciding what a given number of matches means is left to the caller.
//
// Only the first page is fetched. Callers need to distinguish "none", "exactly one",
// and "more than one", and a single page of searchIdentitiesPageSize answers that;
// TotalCount reports the true total when it exceeds the page.
//
// The search endpoint omits identity metadata and auth methods, so callers that need the full record
// should re-fetch the resolved ID with GetIdentity.
func (client Client) SearchIdentitiesByName(name string) (SearchIdentitiesResponse, error) {
	var body SearchIdentitiesResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(SearchIdentitiesRequest{
			Limit:  searchIdentitiesPageSize,
			Offset: 0,
			Search: SearchIdentitiesFilter{
				Name: &SearchIdentitiesNameFilter{Eq: name},
			},
		}).
		Post("api/v1/identities/search")

	if err != nil {
		return SearchIdentitiesResponse{}, errors.NewGenericRequestError(operationSearchIdentities, err)
	}

	if response.IsError() {
		return SearchIdentitiesResponse{}, errors.NewAPIErrorWithResponse(operationSearchIdentities, response, nil)
	}

	return body, nil
}

func (client Client) CreateIdentity(request CreateIdentityRequest) (CreateIdentityResponse, error) {
	var body CreateIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/identities")

	if err != nil {
		return CreateIdentityResponse{}, errors.NewGenericRequestError(operationCreateIdentity, err)
	}

	if response.IsError() {
		return CreateIdentityResponse{}, errors.NewAPIErrorWithResponse(operationCreateIdentity, response, nil)
	}

	return body, nil
}

func (client Client) UpdateIdentity(request UpdateIdentityRequest) (UpdateIdentityResponse, error) {
	var body UpdateIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/identities/" + request.IdentityID)

	if err != nil {
		return UpdateIdentityResponse{}, errors.NewGenericRequestError(operationUpdateIdentity, err)
	}

	if response.IsError() {
		return UpdateIdentityResponse{}, errors.NewAPIErrorWithResponse(operationUpdateIdentity, response, nil)
	}

	return body, nil
}

func (client Client) DeleteIdentity(request DeleteIdentityRequest) (DeleteIdentityResponse, error) {
	var body DeleteIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/identities/" + request.IdentityID)

	if err != nil {
		return DeleteIdentityResponse{}, errors.NewGenericRequestError(operationDeleteIdentity, err)
	}

	if response.IsError() {
		return DeleteIdentityResponse{}, errors.NewAPIErrorWithResponse(operationDeleteIdentity, response, nil)
	}

	return body, nil
}
