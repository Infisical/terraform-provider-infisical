package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

// The Gateway-based Microsoft ADCS CA (CA type "adcs"). The older Azure ADCS web-enrollment CA
// lives under "azure-ad-cs" and is handled by the *ADCSCA functions in certificate_authority.go.
const microsoftADCSCAPath = "api/v1/cert-manager/ca/adcs"

const (
	operationCreateMicrosoftADCSCA = "CallCreateMicrosoftADCSCA"
	operationGetMicrosoftADCSCA    = "CallGetMicrosoftADCSCA"
	operationUpdateMicrosoftADCSCA = "CallUpdateMicrosoftADCSCA"
	operationDeleteMicrosoftADCSCA = "CallDeleteMicrosoftADCSCA"
)

type MicrosoftADCSCAConfiguration struct {
	AppConnectionId string `json:"appConnectionId"`
	// Omitted when nil: the API then discovers the CA name from the CA host through the gateway.
	CaName *string `json:"caName,omitempty"`
}

type MicrosoftADCSCA struct {
	Id            string                       `json:"id"`
	ProjectId     string                       `json:"projectId"`
	Name          string                       `json:"name"`
	Status        string                       `json:"status"`
	Configuration MicrosoftADCSCAConfiguration `json:"configuration"`
}

type CreateMicrosoftADCSCARequest struct {
	Name          string                       `json:"name"`
	Status        string                       `json:"status"`
	Configuration MicrosoftADCSCAConfiguration `json:"configuration"`
}

type UpdateMicrosoftADCSCARequest struct {
	CAId   string `json:"-"`
	Name   string `json:"name"`
	Status string `json:"status"`
	// Nil leaves the connection and CA name as they are.
	Configuration *MicrosoftADCSCAConfiguration `json:"configuration,omitempty"`
}

func (client Client) CreateMicrosoftADCSCA(request CreateMicrosoftADCSCARequest) (MicrosoftADCSCA, error) {
	var ca MicrosoftADCSCA
	response, err := client.Config.HttpClient.
		R().
		SetResult(&ca).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(microsoftADCSCAPath)

	if err != nil {
		return MicrosoftADCSCA{}, errors.NewGenericRequestError(operationCreateMicrosoftADCSCA, err)
	}

	if response.IsError() {
		return MicrosoftADCSCA{}, errors.NewAPIErrorWithResponse(operationCreateMicrosoftADCSCA, response, nil)
	}

	return ca, nil
}

func (client Client) GetMicrosoftADCSCA(caId string) (MicrosoftADCSCA, error) {
	var ca MicrosoftADCSCA
	response, err := client.Config.HttpClient.
		R().
		SetResult(&ca).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("%s/%s", microsoftADCSCAPath, caId))

	if err != nil {
		return MicrosoftADCSCA{}, errors.NewGenericRequestError(operationGetMicrosoftADCSCA, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 {
			return MicrosoftADCSCA{}, ErrNotFound
		}
		return MicrosoftADCSCA{}, errors.NewAPIErrorWithResponse(operationGetMicrosoftADCSCA, response, nil)
	}

	return ca, nil
}

func (client Client) UpdateMicrosoftADCSCA(request UpdateMicrosoftADCSCARequest) (MicrosoftADCSCA, error) {
	var ca MicrosoftADCSCA
	response, err := client.Config.HttpClient.
		R().
		SetResult(&ca).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("%s/%s", microsoftADCSCAPath, request.CAId))

	if err != nil {
		return MicrosoftADCSCA{}, errors.NewGenericRequestError(operationUpdateMicrosoftADCSCA, err)
	}

	if response.IsError() {
		return MicrosoftADCSCA{}, errors.NewAPIErrorWithResponse(operationUpdateMicrosoftADCSCA, response, nil)
	}

	return ca, nil
}

func (client Client) DeleteMicrosoftADCSCA(caId string) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s", microsoftADCSCAPath, caId))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteMicrosoftADCSCA, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeleteMicrosoftADCSCA, response, nil)
	}

	return nil
}
