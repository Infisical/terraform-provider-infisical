package infisicalclient

import (
	"fmt"
	"net/http"

	"terraform-provider-infisical/internal/errors"
)

const (
	operationListGateways               = "CallListGateways"
	operationCreateGateway              = "CallCreateGateway"
	operationGetGateway                 = "CallGetGateway"
	operationUpdateGateway              = "CallUpdateGateway"
	operationDeleteGateway              = "CallDeleteGateway"
	operationMintGatewayEnrollmentToken = "CallMintGatewayEnrollmentToken"
)

// ListGateways returns the gateways in the machine identity's organization. This stays on v2 so the
// gateway data source keeps working against instances that predate the v3 gateway API.
func (client Client) ListGateways() ([]Gateway, error) {
	var gateways []Gateway
	response, err := client.Config.HttpClient.
		R().
		SetResult(&gateways).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v2/gateways")

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListGateways, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationListGateways, response, nil)
	}

	return gateways, nil
}

// GetGatewayByName resolves a gateway by name. Names are unique per organization, so a name
// identifies at most one gateway. ErrNotFound is returned only when the list was retrieved and holds
// no match, since a list that never arrived cannot prove the gateway's absence.
func (client Client) GetGatewayByName(name string) (Gateway, error) {
	gateways, err := client.ListGateways()
	if err != nil {
		return Gateway{}, err
	}

	for _, gateway := range gateways {
		if gateway.Name == name {
			return gateway, nil
		}
	}

	return Gateway{}, ErrNotFound
}

// GatewayAlreadyExistsError carries the id of the gateway holding the name, so a caller whose
// earlier apply created a gateway but failed before recording it can be told what to import.
type GatewayAlreadyExistsError struct {
	ExistingGatewayID string
	apiError          error
}

func (e *GatewayAlreadyExistsError) Error() string {
	return fmt.Sprintf("gateway %s already uses this name: %v", e.ExistingGatewayID, e.apiError)
}

func (e *GatewayAlreadyExistsError) Unwrap() error {
	return e.apiError
}

func (client Client) CreateGateway(request CreateGatewayRequest) (GatewayDetails, error) {
	var body GatewayDetails
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v3/gateways")

	if err != nil {
		return GatewayDetails{}, errors.NewGenericRequestError(operationCreateGateway, err)
	}

	if response.IsError() {
		apiError := errors.NewAPIErrorWithResponse(operationCreateGateway, response, nil)
		if response.StatusCode() == http.StatusBadRequest {
			if existing, lookupErr := client.GetGatewayByName(request.Name); lookupErr == nil {
				return GatewayDetails{}, &GatewayAlreadyExistsError{ExistingGatewayID: existing.ID, apiError: apiError}
			}
		}
		return GatewayDetails{}, apiError
	}

	return body, nil
}

func (client Client) GetGatewayById(id string) (GatewayDetails, error) {
	var body GatewayDetails
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v3/gateways/%s", id))

	if err != nil {
		return GatewayDetails{}, errors.NewGenericRequestError(operationGetGateway, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound || response.StatusCode() == http.StatusUnprocessableEntity {
			return GatewayDetails{}, ErrNotFound
		}
		return GatewayDetails{}, errors.NewAPIErrorWithResponse(operationGetGateway, response, nil)
	}

	return body, nil
}

func (client Client) UpdateGateway(request UpdateGatewayRequest) (GatewayDetails, error) {
	var body GatewayDetails
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v3/gateways/%s", request.ID))

	if err != nil {
		return GatewayDetails{}, errors.NewGenericRequestError(operationUpdateGateway, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GatewayDetails{}, ErrNotFound
		}
		return GatewayDetails{}, errors.NewAPIErrorWithResponse(operationUpdateGateway, response, nil)
	}

	return body, nil
}

func (client Client) DeleteGateway(id string) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v3/gateways/%s", id))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteGateway, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeleteGateway, response, nil)
	}

	return nil
}

// MintGatewayEnrollmentToken issues the one-time bootstrap token for a token-auth gateway. Minting
// invalidates any token issued earlier for the same gateway.
func (client Client) MintGatewayEnrollmentToken(gatewayId string) (MintGatewayEnrollmentTokenResponse, error) {
	var body MintGatewayEnrollmentTokenResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Post(fmt.Sprintf("api/v3/gateways/%s/token-auth/generate-enrollment-token", gatewayId))

	if err != nil {
		return MintGatewayEnrollmentTokenResponse{}, errors.NewGenericRequestError(operationMintGatewayEnrollmentToken, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return MintGatewayEnrollmentTokenResponse{}, ErrNotFound
		}
		return MintGatewayEnrollmentTokenResponse{}, errors.NewAPIErrorWithResponse(operationMintGatewayEnrollmentToken, response, nil)
	}

	return body, nil
}
