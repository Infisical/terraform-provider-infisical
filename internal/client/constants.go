package infisicalclient

import "errors"

const (
	USER_AGENT                                  = "terraform"
	INFISICAL_MACHINE_IDENTITY_ID_NAME          = "INFISICAL_MACHINE_IDENTITY_ID"
	INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET_NAME = "INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET"
	INFISICAL_UNIVERSAL_AUTH_CLIENT_ID_NAME     = "INFISICAL_UNIVERSAL_AUTH_CLIENT_ID"
	INFISICAL_SERVICE_TOKEN_NAME                = "INFISICAL_SERVICE_TOKEN"
	INFISICAL_HOST_NAME                         = "INFISICAL_HOST"
	INFISICAL_AUTH_JWT_NAME                     = "INFISICAL_AUTH_JWT"
)

var (
	ErrNotFound = errors.New("resource not found")
)
