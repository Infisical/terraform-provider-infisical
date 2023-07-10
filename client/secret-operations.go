package infisicalclient

import (
	"fmt"
	"strings"
)

func (client Client) GetPlainTextSecretsViaServiceToken(secretFolderPath string, envSlug string) ([]SingleEnvironmentVariable, *GetServiceTokenDetailsResponse, error) {
	if client.Config.ServiceToken == "" {
		return nil, nil, fmt.Errorf("service token must be defined to fetch secrets")
	}

	serviceTokenParts := strings.SplitN(client.Config.ServiceToken, ".", 4)
	if len(serviceTokenParts) < 4 {
		return nil, nil, fmt.Errorf("invalid service token entered. Please double check your service token and try again")
	}

	serviceTokenDetails, err := client.CallGetServiceTokenDetailsV2()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get service token details. [err=%v]", err)
	}

	request := GetEncryptedSecretsV3Request{
		WorkspaceId: serviceTokenDetails.Workspace,
		Environment: envSlug,
	}

	if secretFolderPath != "" {
		request.SecretPath = secretFolderPath
	}

	encryptedSecrets, err := client.CallGetSecretsV3(request)

	if err != nil {
		return nil, nil, err
	}

	decodedSymmetricEncryptionDetails, err := GetBase64DecodedSymmetricEncryptionDetails(serviceTokenParts[3], serviceTokenDetails.EncryptedKey, serviceTokenDetails.Iv, serviceTokenDetails.Tag)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decode symmetric encryption details [err=%v]", err)
	}

	plainTextWorkspaceKey, err := DecryptSymmetric([]byte(serviceTokenParts[3]), decodedSymmetricEncryptionDetails.Cipher, decodedSymmetricEncryptionDetails.Tag, decodedSymmetricEncryptionDetails.IV)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decrypt the required workspace key")
	}

	plainTextSecrets, err := GetPlainTextSecrets(plainTextWorkspaceKey, encryptedSecrets)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to decrypt your secrets [err=%v]", err)
	}

	return plainTextSecrets, &serviceTokenDetails, nil
}
