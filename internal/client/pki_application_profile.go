package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationListPkiApplicationProfiles  = "CallListPkiApplicationProfiles"
	operationAttachPkiApplicationProfile = "CallAttachPkiApplicationProfiles"
	operationDetachPkiApplicationProfile = "CallDetachPkiApplicationProfile"
)

func (client Client) ListPkiApplicationProfiles(request ListPkiApplicationProfilesRequest) (ListPkiApplicationProfilesResponse, error) {
	var profilesResponse ListPkiApplicationProfilesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profilesResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles", request.ApplicationId))

	if err != nil {
		return ListPkiApplicationProfilesResponse{}, errors.NewGenericRequestError(operationListPkiApplicationProfiles, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return ListPkiApplicationProfilesResponse{}, ErrNotFound
		}
		return ListPkiApplicationProfilesResponse{}, errors.NewAPIErrorWithResponse(operationListPkiApplicationProfiles, response, nil)
	}

	return profilesResponse, nil
}

func (client Client) AttachPkiApplicationProfiles(request AttachPkiApplicationProfilesRequest) (AttachPkiApplicationProfilesResponse, error) {
	var profilesResponse AttachPkiApplicationProfilesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profilesResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles", request.ApplicationId))

	if err != nil {
		return AttachPkiApplicationProfilesResponse{}, errors.NewGenericRequestError(operationAttachPkiApplicationProfile, err)
	}

	if response.IsError() {
		return AttachPkiApplicationProfilesResponse{}, errors.NewAPIErrorWithResponse(operationAttachPkiApplicationProfile, response, nil)
	}

	return profilesResponse, nil
}

func (client Client) DetachPkiApplicationProfile(request DetachPkiApplicationProfileRequest) (DetachPkiApplicationProfileResponse, error) {
	var profileResponse DetachPkiApplicationProfileResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&profileResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/profiles/%s", request.ApplicationId, request.ProfileId))

	if err != nil {
		return DetachPkiApplicationProfileResponse{}, errors.NewGenericRequestError(operationDetachPkiApplicationProfile, err)
	}

	if response.IsError() {
		return DetachPkiApplicationProfileResponse{}, errors.NewAPIErrorWithResponse(operationDetachPkiApplicationProfile, response, nil)
	}

	return profileResponse, nil
}
