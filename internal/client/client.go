package infisicalclient

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	Config Config
}

type AuthStrategyType string

var AuthStrategy = struct {
	SERVICE_TOKEN               AuthStrategyType
	UNIVERSAL_MACHINE_IDENTITY  AuthStrategyType
	OIDC_MACHINE_IDENTITY       AuthStrategyType
	TOKEN_MACHINE_IDENTITY      AuthStrategyType
	KUBERNETES_MACHINE_IDENTITY AuthStrategyType
}{
	SERVICE_TOKEN:               "SERVICE_TOKEN",
	UNIVERSAL_MACHINE_IDENTITY:  "UNIVERSAL_MACHINE_IDENTITY",
	OIDC_MACHINE_IDENTITY:       "OIDC_MACHINE_IDENTITY",
	TOKEN_MACHINE_IDENTITY:      "TOKEN_MACHINE_IDENTITY",
	KUBERNETES_MACHINE_IDENTITY: "KUBERNETES_MACHINE_IDENTITY",
}

type Config struct {
	HostURL string

	AuthStrategy          AuthStrategyType
	IsMachineIdentityAuth bool

	// Service Token Auth
	ServiceToken string

	// Universal Machine Identity Auth
	ClientId     string
	ClientSecret string

	// Token machine identity auth
	Token string

	//OIDC Machine Identity Auth
	IdentityId       string
	OidcTokenEnvName string

	// Kubernetes Machine Identity Auth
	ServiceAccountToken     string
	ServiceAccountTokenPath string

	EnvSlug     string
	SecretsPath string
	HttpClient  *resty.Client // By default a client will be created
}

func NewClient(cnf Config) (*Client, error) {
	if cnf.HttpClient == nil {
		cnf.HttpClient = resty.New()
		cnf.HttpClient.SetBaseURL(cnf.HostURL)
	}

	var usingServiceToken = cnf.ServiceToken != ""

	selectedAuthStrategy := cnf.AuthStrategy
	if cnf.ClientId != "" && cnf.ClientSecret != "" && selectedAuthStrategy == "" {
		selectedAuthStrategy = AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
	}

	// Check if the user got multiple configured authentication methods, or none set at all.
	if usingServiceToken && selectedAuthStrategy != "" {
		return nil, fmt.Errorf("you have configured multiple authentication methods, please only use one")
	} else if !usingServiceToken && selectedAuthStrategy == "" {
		return nil, fmt.Errorf("you must configure an authentication method such as service tokens or Universal Auth before making calls")
	}

	if usingServiceToken {
		cnf.HttpClient.SetAuthToken(cnf.ServiceToken)
		cnf.AuthStrategy = AuthStrategy.SERVICE_TOKEN
	} else {
		authStrategies := map[AuthStrategyType]func() (string, error){
			AuthStrategy.UNIVERSAL_MACHINE_IDENTITY:  Client{cnf}.UniversalMachineIdentityAuth,
			AuthStrategy.OIDC_MACHINE_IDENTITY:       Client{cnf}.OidcMachineIdentityAuth,
			AuthStrategy.TOKEN_MACHINE_IDENTITY:      Client{cnf}.TokenMachineIdentityAuth,
			AuthStrategy.KUBERNETES_MACHINE_IDENTITY: Client{cnf}.KubernetesMachineIdentityAuth,
		}

		token, err := authStrategies[selectedAuthStrategy]()
		if err != nil {
			return nil, fmt.Errorf("unable to authenticate with machine identity [err=%s]", err)
		}

		cnf.AuthStrategy = selectedAuthStrategy
		cnf.IsMachineIdentityAuth = true
		cnf.HttpClient.SetAuthToken(token)
	}

	// These two if statements were a part of an older migration.
	// And when people upgraded to the newer version, we needed a way to indicate that the EnvSlug and SecretsPath are no longer defined on a provider-level.
	if cnf.EnvSlug != "" {
		return nil, fmt.Errorf("you must set the environment before making calls")
	}

	if cnf.SecretsPath != "" {
		return nil, fmt.Errorf("you must specify the secrets path before making calls")
	}

	cnf.HttpClient.SetHeader("Accept", "application/json")

	return &Client{cnf}, nil
}
