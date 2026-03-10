package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationAddGroupMachineIdentity    = "CallAddGroupMachineIdentity"
	operationListGroupMachineIdentities = "CallListGroupMachineIdentities"
	operationRemoveGroupMachineIdentity = "CallRemoveGroupMachineIdentity"
)

func (client Client) AddGroupMachineIdentity(request AddGroupMachineIdentityRequest) (AddGroupMachineIdentityResponse, error) {
	var responseData AddGroupMachineIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Post(fmt.Sprintf("api/v1/groups/%s/machine-identities/%s", request.GroupID, request.IdentityID))

	if err != nil {
		return AddGroupMachineIdentityResponse{}, errors.NewGenericRequestError(operationAddGroupMachineIdentity, err)
	}

	if response.IsError() {
		return AddGroupMachineIdentityResponse{}, errors.NewAPIErrorWithResponse(operationAddGroupMachineIdentity, response, nil)
	}

	return responseData, nil
}

func (client Client) ListGroupMachineIdentities(request ListGroupMachineIdentitiesRequest) (ListGroupMachineIdentitiesResponse, error) {
	var responseData ListGroupMachineIdentitiesResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/groups/%s/machine-identities", request.GroupID))

	if response.StatusCode() == http.StatusNotFound {
		return ListGroupMachineIdentitiesResponse{}, ErrNotFound
	}

	if err != nil {
		return ListGroupMachineIdentitiesResponse{}, errors.NewGenericRequestError(operationListGroupMachineIdentities, err)
	}

	if response.IsError() {
		return ListGroupMachineIdentitiesResponse{}, errors.NewAPIErrorWithResponse(operationListGroupMachineIdentities, response, nil)
	}

	return responseData, nil
}

func (client Client) RemoveGroupMachineIdentity(request RemoveGroupMachineIdentityRequest) (RemoveGroupMachineIdentityResponse, error) {
	var responseData RemoveGroupMachineIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/groups/%s/machine-identities/%s", request.GroupID, request.IdentityID))

	if response.StatusCode() == http.StatusNotFound {
		return RemoveGroupMachineIdentityResponse{}, ErrNotFound
	}

	if err != nil {
		return RemoveGroupMachineIdentityResponse{}, errors.NewGenericRequestError(operationRemoveGroupMachineIdentity, err)
	}

	if response.IsError() {
		return RemoveGroupMachineIdentityResponse{}, errors.NewAPIErrorWithResponse(operationRemoveGroupMachineIdentity, response, nil)
	}

	return responseData, nil
}
