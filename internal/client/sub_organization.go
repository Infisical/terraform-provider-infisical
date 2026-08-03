package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateSubOrganization = "CallCreateSubOrganization"
	operationListSubOrganizations  = "CallListSubOrganizations"
	operationUpdateSubOrganization = "CallUpdateSubOrganization"
	operationDeleteSubOrganization = "CallDeleteSubOrganization"
)

func (client Client) CreateSubOrganization(request CreateSubOrganizationRequest) (CreateSubOrganizationResponse, error) {
	var responseData CreateSubOrganizationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/sub-organizations")

	if err != nil {
		return CreateSubOrganizationResponse{}, errors.NewGenericRequestError(operationCreateSubOrganization, err)
	}

	if response.IsError() {
		return CreateSubOrganizationResponse{}, errors.NewAPIErrorWithResponse(operationCreateSubOrganization, response, nil)
	}

	return responseData, nil
}

// ListSubOrganizations returns every sub-organization under the caller's root org
// (subject to root-org RBAC). It paginates over the offset until the full set is
// retrieved. isAccessible is intentionally NOT set so that sub-orgs the caller is
// not a member of are still returned, which keeps Read and Import robust.
func (client Client) ListSubOrganizations() ([]SubOrganization, error) {
	const pageSize = 1000
	offset := 0
	var allSubOrgs []SubOrganization

	for {
		var responseData ListSubOrganizationsResponse
		response, err := client.Config.HttpClient.
			R().
			SetResult(&responseData).
			SetHeader("User-Agent", USER_AGENT).
			SetQueryParams(map[string]string{
				"limit":  fmt.Sprintf("%d", pageSize),
				"offset": fmt.Sprintf("%d", offset),
			}).
			Get("api/v1/sub-organizations")

		if err != nil {
			return nil, errors.NewGenericRequestError(operationListSubOrganizations, err)
		}

		if response.IsError() {
			return nil, errors.NewAPIErrorWithResponse(operationListSubOrganizations, response, nil)
		}

		allSubOrgs = append(allSubOrgs, responseData.Organizations...)
		offset += pageSize

		if len(responseData.Organizations) == 0 || offset >= responseData.TotalCount {
			break
		}
	}

	return allSubOrgs, nil
}

// GetSubOrganizationById resolves a single sub-organization by ID. The API exposes
// no GET-by-id endpoint, so this lists sub-orgs and filters client-side. It returns
// ErrNotFound when no sub-org with the given ID exists.
func (client Client) GetSubOrganizationById(id string) (SubOrganization, error) {
	subOrgs, err := client.ListSubOrganizations()
	if err != nil {
		return SubOrganization{}, err
	}

	for _, subOrg := range subOrgs {
		if subOrg.ID == id {
			return subOrg, nil
		}
	}

	return SubOrganization{}, ErrNotFound
}

func (client Client) UpdateSubOrganization(request UpdateSubOrganizationRequest) (UpdateSubOrganizationResponse, error) {
	var responseData UpdateSubOrganizationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/sub-organizations/%s", request.SubOrgID))

	if err != nil {
		return UpdateSubOrganizationResponse{}, errors.NewGenericRequestError(operationUpdateSubOrganization, err)
	}

	if response.IsError() {
		return UpdateSubOrganizationResponse{}, errors.NewAPIErrorWithResponse(operationUpdateSubOrganization, response, nil)
	}

	return responseData, nil
}

func (client Client) DeleteSubOrganization(request DeleteSubOrganizationRequest) (DeleteSubOrganizationResponse, error) {
	var responseData DeleteSubOrganizationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/sub-organizations/%s", request.SubOrgID))

	if err != nil {
		return DeleteSubOrganizationResponse{}, errors.NewGenericRequestError(operationDeleteSubOrganization, err)
	}

	if response.IsError() {
		return DeleteSubOrganizationResponse{}, errors.NewAPIErrorWithResponse(operationDeleteSubOrganization, response, nil)
	}

	return responseData, nil
}
