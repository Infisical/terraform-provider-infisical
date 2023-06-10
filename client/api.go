package infisicalclient

import "fmt"

const USER_AGENT = "terraform"

func (client Client) CallGetServiceTokenDetailsV2() (GetServiceTokenDetailsResponse, error) {
	var tokenDetailsResponse GetServiceTokenDetailsResponse
	response, err := client.cnf.HttpClient.
		R().
		SetResult(&tokenDetailsResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v2/service-token")

	fmt.Println("response===>", response.Request)

	if err != nil {
		return GetServiceTokenDetailsResponse{}, fmt.Errorf("CallGetServiceTokenDetails: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetServiceTokenDetailsResponse{}, fmt.Errorf("CallGetServiceTokenDetails: Unsuccessful response: [response=%s]", response)
	}

	return tokenDetailsResponse, nil
}

func (client Client) CallGetSecretsV2(request GetEncryptedSecretsV2Request) (GetEncryptedSecretsV2Response, error) {
	var secretsResponse GetEncryptedSecretsV2Response
	requestToBeMade := client.cnf.HttpClient.
		R().
		SetResult(&secretsResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("workspaceId", request.WorkspaceId).
		SetQueryParam("tagSlugs", request.TagSlugs)

	if request.SecretPath != "" {
		requestToBeMade.SetQueryParam("secretsPath", request.SecretPath)
	}

	response, err := requestToBeMade.
		Get("api/v2/secrets")

	if err != nil {
		return GetEncryptedSecretsV2Response{}, fmt.Errorf("CallGetSecretsV2: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetEncryptedSecretsV2Response{}, fmt.Errorf("CallGetSecretsV2: Unsuccessful response: [response=%v]", response.RawResponse)
	}

	return secretsResponse, nil
}
