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

	// Add more auth strategies here later
	var usingServiceToken = cnf.ServiceToken != ""
	var usingUniversalAuth = cnf.ClientId != "" && cnf.ClientSecret != ""

	// Check if the user got multiple configured authentication methods, or none set at all.
	if usingServiceToken && usingUniversalAuth {
		return nil, fmt.Errorf("you have configured multiple authentication methods, please only use one")
	} else if !usingServiceToken && !usingUniversalAuth {
		return nil, fmt.Errorf("you must configure a authentication method such as service tokens or Universal Auth before making calls")
	}

	if usingUniversalAuth {
		token, err := Client{cnf}.UniversalMachineIdentityAuth()

		if err != nil {
			return nil, fmt.Errorf("unable to authenticate with universal machine identity [err=%s]", err)
		}

		cnf.HttpClient.SetAuthToken(token)
		cnf.AuthStrategy = AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
	} else if usingServiceToken {
		cnf.HttpClient.SetAuthToken(cnf.ServiceToken)
		cnf.AuthStrategy = AuthStrategy.SERVICE_TOKEN
	} else {
		// If no auth strategy is set, then we should return an error
		return nil, fmt.Errorf("you must configure a authentication method such as service tokens or Universal Auth before making calls")
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
