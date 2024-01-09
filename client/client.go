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
	SERVICE_TOKEN              AuthStrategyType
	UNIVERSAL_MACHINE_IDENTITY AuthStrategyType
}{
	SERVICE_TOKEN:              "SERVICE_TOKEN",
	UNIVERSAL_MACHINE_IDENTITY: "UNIVERSAL_MACHINE_IDENTITY",
}

type Config struct {
	HostURL string

	AuthStrategy AuthStrategyType

	// Service Token Auth
	ServiceToken string

	// Universal Machine Identity Auth
	ClientId     string
	ClientSecret string

	EnvSlug     string
	SecretsPath string
	HttpClient  *resty.Client // By default a client will be created
}

func NewClient(cnf Config) (*Client, error) {
	if cnf.HttpClient == nil {
		cnf.HttpClient = resty.New()
		cnf.HttpClient.SetBaseURL(cnf.HostURL)
	}

	if cnf.ServiceToken == "" && cnf.ClientId == "" && cnf.ClientSecret == "" {
		return nil, fmt.Errorf("you must set the service token, or a client secret and client ID for the client before making calls")
	}

	var authToken string

	if cnf.ClientId != "" && cnf.ClientSecret != "" {
		token, err := Client{cnf}.UniversalMachineIdentityAuth()

		if err != nil {
			return nil, fmt.Errorf("unable to authenticate with universal machine identity [err=%s]", err)
		}

		authToken = token
		cnf.AuthStrategy = AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
	}
	if cnf.ServiceToken != "" && authToken == "" {
		authToken = cnf.ServiceToken
		cnf.AuthStrategy = AuthStrategy.SERVICE_TOKEN
	}

	if authToken != "" {
		cnf.HttpClient.SetAuthToken(authToken)
	} else {
		return nil, fmt.Errorf("no authentication credentials provided. You must define the service_token, or client_id and client_secret field of the provider")
	}
	if cnf.EnvSlug != "" {
		return nil, fmt.Errorf("you must set the environment before making calls")
	}

	if cnf.SecretsPath != "" {
		return nil, fmt.Errorf("you must specify the secrets path before making calls")
	}

	cnf.HttpClient.SetHeader("Accept", "application/json")

	return &Client{cnf}, nil
}
