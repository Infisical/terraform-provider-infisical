package infisicalclient

import (
	"terraform-provider-infisical/internal/errors"
)

const operationListGateways = "CallListGateways"

// ListGateways returns the gateways in the machine identity's organization.
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
