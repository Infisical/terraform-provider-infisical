package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateEmailDomain = "CallCreateEmailDomain"
	operationListEmailDomains  = "CallListEmailDomains"
	operationDeleteEmailDomain = "CallDeleteEmailDomain"
)

func (client Client) CreateEmailDomain(request CreateEmailDomainRequest) (EmailDomain, error) {
	var responseData CreateEmailDomainResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/email-domains")

	if err != nil {
		return EmailDomain{}, errors.NewGenericRequestError(operationCreateEmailDomain, err)
	}

	if response.IsError() {
		return EmailDomain{}, errors.NewAPIErrorWithResponse(operationCreateEmailDomain, response, nil)
	}

	return responseData.EmailDomain, nil
}

func (client Client) ListEmailDomains() ([]EmailDomain, error) {
	var responseData ListEmailDomainsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/email-domains")

	if err != nil {
		return nil, errors.NewGenericRequestError(operationListEmailDomains, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationListEmailDomains, response, nil)
	}

	return responseData.EmailDomains, nil
}

// API has no get by ID endpoint, hence using ListEmailDomains
func (client Client) GetEmailDomainByID(request GetEmailDomainByIDRequest) (EmailDomain, error) {
	emailDomains, err := client.ListEmailDomains()
	if err != nil {
		return EmailDomain{}, err
	}

	for _, emailDomain := range emailDomains {
		if emailDomain.ID == request.ID {
			return emailDomain, nil
		}
	}

	return EmailDomain{}, ErrNotFound
}

func (client Client) DeleteEmailDomain(request DeleteEmailDomainRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/email-domains/%s", request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteEmailDomain, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteEmailDomain, response, nil)
	}

	return nil
}
