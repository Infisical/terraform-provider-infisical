package pkg

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ValidateAwsInputCredentials(accessKeyId basetypes.StringValue, secretAccessKey basetypes.StringValue, assumeRoleArn basetypes.StringValue) (AwsAuthenticationMethod, error) {

	// No credentials provided at all
	if assumeRoleArn.ValueString() == "" && (accessKeyId.ValueString() == "" || secretAccessKey.ValueString() == "") {
		return "", fmt.Errorf("No credentials provided. Either set access_key_id and secret_access_key, or assume_role_arn.")
	}

	if accessKeyId.ValueString() != "" && secretAccessKey.ValueString() != "" && assumeRoleArn.ValueString() != "" {
		return "", fmt.Errorf("Both access_key_id and secret_access_key, and assume_role_arn are provided. Only one set of credentials can be used. Either set access_key_id and secret_access_key, or assume_role_arn.")
	}

	// Access key and secret key provided
	if accessKeyId.ValueString() != "" && secretAccessKey.ValueString() != "" {
		return AwsAuthMethodAccessKey, nil
	}

	// Assume role provided
	if assumeRoleArn.ValueString() != "" {
		return AwsAuthMethodAssumeRole, nil
	}

	return "", fmt.Errorf("Invalid credentials provided. Either set access_key_id and secret_access_key, or assume_role_arn.")
}
